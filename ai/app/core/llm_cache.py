import json
import hashlib
import logging
from typing import Dict, Any, List, Optional
import numpy as np
import redis

from app.core.config import settings

logger = logging.getLogger("ai-service.core.llm_cache")


class LLMSemanticCache:
    """
    2-Tier LLM Response Semantic Cache (GPTCache Pattern)
    - Tier 1: Exact Query SHA-256 Hash Matching (< 0.5ms)
    - Tier 2: Semantic Vector Cosine Similarity Matching with bge-m3 (< 5ms)
    """

    def __init__(self, redis_client: Optional[redis.Redis] = None):
        self._redis = redis_client
        if self._redis is None:
            self._init_redis()

    def _init_redis(self):
        try:
            self._redis = redis.Redis(
                host=settings.REDIS_HOST,
                port=settings.REDIS_PORT,
                password=settings.REDIS_PASSWORD if settings.REDIS_PASSWORD else None,
                db=settings.REDIS_DB,
                decode_responses=True,
                socket_timeout=2.0,
                socket_connect_timeout=2.0
            )
            self._redis.ping()
            logger.info(f"✅ [LLM_CACHE] Connected to Redis LLM Semantic Cache at {settings.REDIS_HOST}:{settings.REDIS_PORT}")
        except Exception as e:
            logger.warning(f"⚠️ [LLM_CACHE] Redis connection unavailable ({e}). Running without LLM Semantic Cache.")
            self._redis = None

    def get_cached_turn(
        self,
        query: str,
        query_vector: Optional[List[float]] = None,
        scope: str = "shopper",
        threshold: Optional[float] = None
    ) -> Optional[Dict[str, Any]]:
        """
        Check Tier 1 (Exact Hash) and Tier 2 (Semantic Vector) caches in Redis.
        Returns cached response payload if hit, otherwise None.
        """
        if not self._redis or not settings.LLM_CACHE_ENABLED:
            return None

        clean_query = query.strip()
        if not clean_query:
            return None

        sim_threshold = threshold if threshold is not None else settings.LLM_CACHE_SEMANTIC_THRESHOLD

        # ----------------------------------------------------------------------
        # Tier 1: Exact Hash Cache (< 0.5ms)
        # ----------------------------------------------------------------------
        try:
            hash_val = hashlib.sha256(clean_query.lower().encode('utf-8')).hexdigest()[:16]
            exact_key = f"llm:exact:{scope}:{hash_val}"
            raw_exact = self._redis.get(exact_key)
            if raw_exact:
                cached_data = json.loads(raw_exact)
                logger.info(
                    f"⚡ [LLM_CACHE_HIT: EXACT] scope='{scope}' query='{clean_query}' "
                    f"| Tools: {len(cached_data.get('tool_calls', []))} | Prods: {len(cached_data.get('suggested_products', []))}"
                )
                return cached_data
        except Exception as e:
            logger.warning(f"Error reading exact LLM cache: {e}")

        # ----------------------------------------------------------------------
        # Tier 2: Semantic Vector Cosine Matching (< 5ms)
        # ----------------------------------------------------------------------
        if query_vector is None:
            return None

        try:
            semantic_key = f"llm:semantic:{scope}"
            raw_entries = self._redis.lrange(semantic_key, 0, -1)
            if not raw_entries:
                return None

            q_np = np.array(query_vector, dtype=np.float32)
            q_norm = np.linalg.norm(q_np)
            if q_norm == 0:
                return None

            best_sim = -1.0
            best_entry: Optional[Dict[str, Any]] = None

            for raw in raw_entries:
                try:
                    entry = json.loads(raw)
                    cached_vec = np.array(entry["vector"], dtype=np.float32)
                    c_norm = np.linalg.norm(cached_vec)
                    if c_norm == 0:
                        continue
                    sim = float(np.dot(q_np, cached_vec) / (q_norm * c_norm))
                    if sim > best_sim:
                        best_sim = sim
                        best_entry = entry
                except Exception:
                    continue

            if best_sim >= sim_threshold and best_entry:
                cached_payload = best_entry.get("payload", {})
                matched_query = best_entry.get("query", "")
                logger.info(
                    f"🎯 [LLM_CACHE_HIT: SEMANTIC ({best_sim * 100:.1f}% >= {sim_threshold * 100:.0f}%)] "
                    f"query='{clean_query}' ~ matched='{matched_query}'"
                )
                # Save to exact hash cache for instant future hits
                hash_val = hashlib.sha256(clean_query.lower().encode('utf-8')).hexdigest()[:16]
                exact_key = f"llm:exact:{scope}:{hash_val}"
                self._redis.setex(exact_key, settings.LLM_CACHE_EXACT_TTL_SECONDS, json.dumps(cached_payload))
                return cached_payload

        except Exception as e:
            logger.warning(f"Error reading semantic LLM cache: {e}")

        return None

    def set_cached_turn(
        self,
        query: str,
        query_vector: List[float],
        payload: Dict[str, Any],
        scope: str = "shopper"
    ):
        """
        Store agent output turn into Tier 1 Exact Hash and Tier 2 Semantic Vector caches.
        """
        if not self._redis or not settings.LLM_CACHE_ENABLED:
            return

        clean_query = query.strip()
        if not clean_query or not payload:
            return

        try:
            # 1. Store Tier 1 Exact Hash
            hash_val = hashlib.sha256(clean_query.lower().encode('utf-8')).hexdigest()[:16]
            exact_key = f"llm:exact:{scope}:{hash_val}"
            self._redis.setex(
                exact_key,
                settings.LLM_CACHE_EXACT_TTL_SECONDS,
                json.dumps(payload, ensure_ascii=False)
            )

            # 2. Store Tier 2 Semantic Vector List
            semantic_key = f"llm:semantic:{scope}"
            entry = {
                "query": clean_query,
                "vector": query_vector,
                "payload": payload,
            }

            pipe = self._redis.pipeline()
            pipe.rpush(semantic_key, json.dumps(entry, ensure_ascii=False))
            if settings.LLM_CACHE_MAX_ENTRIES > 0:
                pipe.ltrim(semantic_key, -settings.LLM_CACHE_MAX_ENTRIES, -1)
            pipe.expire(semantic_key, settings.LLM_CACHE_SEMANTIC_TTL_SECONDS)
            pipe.execute()

            logger.info(f"💾 [LLM_CACHE_SAVED] scope='{scope}' query='{clean_query}'")
        except Exception as e:
            logger.warning(f"Error saving to LLM cache: {e}")

    def invalidate(self, scope: Optional[str] = None):
        """
        Invalidate cached conversational turns across exact and semantic scopes.
        """
        if not self._redis:
            return

        try:
            pattern_exact = f"llm:exact:{scope}:*" if scope else "llm:exact:*"
            pattern_semantic = f"llm:semantic:{scope}" if scope else "llm:semantic:*"

            exact_keys = list(self._redis.scan_iter(pattern_exact))
            semantic_keys = list(self._redis.scan_iter(pattern_semantic))
            all_keys = exact_keys + semantic_keys

            if all_keys:
                self._redis.delete(*all_keys)
                logger.info(f"🧹 [LLM_CACHE_INVALIDATED] Cleared {len(all_keys)} LLM response cache keys (scope={scope or 'ALL'}).")
        except Exception as e:
            logger.warning(f"Error invalidating LLM cache: {e}")
