import io
import re
import json
import hashlib
import logging
from typing import List, Dict, Any, Optional

import pypdf
import numpy as np
import redis

from app.core.config import settings
from app.repositories.embedding_repository import EmbeddingRepository
from app.repositories.knowledge_repository import KnowledgeRepository

logger = logging.getLogger("ai-service.usecase.knowledge")


class KnowledgeUseCase:
    """UseCase managing in-memory PDF parsing, text chunking, embedding generation, pgvector retrieval, and Redis Semantic Caching"""

    def __init__(
        self,
        knowledge_repo: KnowledgeRepository,
        embedding_repo: EmbeddingRepository
    ):
        self.knowledge_repo = knowledge_repo
        self.embedding_repo = embedding_repo
        self._redis: Optional[redis.Redis] = None
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
            logger.info(f"✅ [REDIS] Connected to Redis RAG Cache at {settings.REDIS_HOST}:{settings.REDIS_PORT}")
        except Exception as e:
            logger.warning(f"⚠️ [REDIS] Failed to connect to Redis for RAG caching ({e}). Running RAG without cache.")
            self._redis = None

    def _chunk_text(
        self,
        text: str,
        page_number: int,
        chunk_size: int = 500,
        chunk_overlap: int = 100
    ) -> List[Dict[str, Any]]:
        """Split text into semantically cohesive overlapping chunks without breaking mid-sentence where possible"""
        clean_text = re.sub(r'[ \t]+', ' ', text).strip()
        if not clean_text:
            return []

        paragraphs = [p.strip() for p in clean_text.split('\n') if p.strip()]
        chunks = []
        current_chunk = ""

        for para in paragraphs:
            if len(current_chunk) + len(para) <= chunk_size:
                current_chunk += ("\n" if current_chunk else "") + para
            else:
                if current_chunk:
                    chunks.append(current_chunk)
                overlap_text = current_chunk[-chunk_overlap:] if len(current_chunk) > chunk_overlap else ""
                current_chunk = (overlap_text + "\n" + para).strip() if overlap_text else para

        if current_chunk:
            chunks.append(current_chunk)

        final_chunks = []
        for c in chunks:
            if len(c) > chunk_size * 1.5:
                sentences = re.split(r'(?<=[.?!])\s+', c)
                sub_c = ""
                for s in sentences:
                    if len(sub_c) + len(s) <= chunk_size:
                        sub_c += (" " if sub_c else "") + s
                    else:
                        if sub_c:
                            final_chunks.append(sub_c)
                        sub_c = s
                if sub_c:
                    final_chunks.append(sub_c)
            else:
                final_chunks.append(c)

        return [
            {
                "content": fc.strip(),
                "page_number": page_number
            }
            for fc in final_chunks if len(fc.strip()) > 15
        ]

    def _get_exact_cache(self, clean_query: str, scope: str) -> Optional[List[Dict[str, Any]]]:
        """Check Level 1 Exact Hash Cache in Redis (< 0.5ms)"""
        if not self._redis or not settings.RAG_CACHE_ENABLED:
            return None
        try:
            hash_val = hashlib.sha256(clean_query.lower().encode('utf-8')).hexdigest()[:16]
            key = f"rag:exact:{scope}:{hash_val}"
            raw = self._redis.get(key)
            if raw:
                logger.info(f"⚡ [REDIS RAG CACHE HIT: Exact Match] scope='{scope}' query='{clean_query}'")
                return json.loads(raw)
        except Exception as e:
            logger.warning(f"Error reading exact RAG cache: {e}")
        return None

    def _get_semantic_cache(
        self,
        clean_query: str,
        query_vec: List[float],
        scope: str,
        threshold: float
    ) -> Optional[List[Dict[str, Any]]]:
        """Check Level 2 Semantic Vector Similarity Cache in Redis (< 5ms)"""
        if not self._redis or not settings.RAG_CACHE_ENABLED:
            return None
        try:
            key = f"rag:semantic:{scope}"
            raw_entries = self._redis.lrange(key, 0, -1)
            if not raw_entries:
                return None

            q_np = np.array(query_vec, dtype=np.float32)
            q_norm = np.linalg.norm(q_np)
            if q_norm == 0:
                return None

            best_sim = -1.0
            best_results = None
            matched_query = ""

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
                        best_results = entry.get("results")
                        matched_query = entry.get("query", "")
                except Exception:
                    continue

            if best_sim >= threshold and best_results:
                logger.info(f"🎯 [REDIS RAG CACHE HIT: Semantic Match ({best_sim * 100:.1f}%)] query='{clean_query}' ~ matched='{matched_query}'")
                # Also save to exact cache for instant future hits
                hash_val = hashlib.sha256(clean_query.lower().encode('utf-8')).hexdigest()[:16]
                exact_key = f"rag:exact:{scope}:{hash_val}"
                self._redis.setex(exact_key, settings.RAG_CACHE_TTL_SECONDS, json.dumps(best_results))
                return best_results
        except Exception as e:
            logger.warning(f"Error reading semantic RAG cache: {e}")
        return None

    def _save_to_cache(
        self,
        clean_query: str,
        query_vec: List[float],
        scope: str,
        results: List[Dict[str, Any]]
    ):
        """Save query and chunks to Redis Exact and Semantic Vector caches"""
        if not self._redis or not settings.RAG_CACHE_ENABLED or not results:
            return
        try:
            # 1. Exact Cache
            hash_val = hashlib.sha256(clean_query.lower().encode('utf-8')).hexdigest()[:16]
            exact_key = f"rag:exact:{scope}:{hash_val}"
            self._redis.setex(exact_key, settings.RAG_CACHE_TTL_SECONDS, json.dumps(results))

            # 2. Semantic Vector List Cache
            semantic_key = f"rag:semantic:{scope}"
            payload = json.dumps({
                "query": clean_query,
                "vector": query_vec,
                "results": results
            }, ensure_ascii=False)

            pipe = self._redis.pipeline()
            pipe.rpush(semantic_key, payload)
            if settings.RAG_CACHE_MAX_ENTRIES > 0:
                pipe.ltrim(semantic_key, -settings.RAG_CACHE_MAX_ENTRIES, -1)
            pipe.expire(semantic_key, settings.RAG_CACHE_TTL_SECONDS)
            pipe.execute()
            logger.info(f"💾 [REDIS RAG CACHE SAVED] scope='{scope}' query='{clean_query}' (results={len(results)})")
        except Exception as e:
            logger.warning(f"Error saving to RAG cache: {e}")

    def invalidate_cache(self):
        """Flush all RAG exact and semantic caches from Redis upon document changes"""
        if not self._redis:
            return
        try:
            exact_keys = list(self._redis.scan_iter("rag:exact:*"))
            semantic_keys = list(self._redis.scan_iter("rag:semantic:*"))
            all_keys = exact_keys + semantic_keys
            if all_keys:
                self._redis.delete(*all_keys)
                logger.info(f"🧹 [REDIS RAG CACHE INVALIDATED] Cleared {len(all_keys)} cached RAG keys.")
        except Exception as e:
            logger.warning(f"Error invalidating RAG cache: {e}")

    def index_pdf_in_memory(
        self,
        file_bytes: bytes,
        filename: str,
        title: Optional[str] = None,
        doc_type: str = "GENERAL"
    ) -> Dict[str, Any]:
        """Parse uploaded PDF file entirely in-memory, extract pages, compute vector embeddings, and save to database"""
        logger.info(f"📄 Processing PDF in-memory: filename='{filename}' | size={len(file_bytes)} bytes | doc_type='{doc_type}'")

        pdf_stream = io.BytesIO(file_bytes)
        reader = pypdf.PdfReader(pdf_stream)
        total_pages = len(reader.pages)

        raw_chunks: List[Dict[str, Any]] = []
        for page_idx, page in enumerate(reader.pages):
            page_num = page_idx + 1
            page_text = page.extract_text() or ""
            page_chunks = self._chunk_text(page_text, page_number=page_num)
            raw_chunks.extend(page_chunks)

        if not raw_chunks:
            raise ValueError(f"No readable text could be extracted from PDF '{filename}'. Please ensure the PDF is not an image-only scan.")

        for idx, rc in enumerate(raw_chunks):
            rc["chunk_index"] = idx

        texts_to_embed = [c["content"] for c in raw_chunks]
        logger.info(f"🧠 Computing dense embeddings for {len(texts_to_embed)} chunks from '{filename}'...")
        embeddings = self.embedding_repo.encode_batch(texts_to_embed)

        for chunk, emb in zip(raw_chunks, embeddings):
            chunk["embedding"] = emb

        doc_title = title or filename.replace(".pdf", "").replace("_", " ").title()
        saved_doc = self.knowledge_repo.save_document_and_chunks(
            title=doc_title,
            doc_type=doc_type.upper(),
            filename=filename,
            total_pages=total_pages,
            chunks=raw_chunks
        )

        # Invalidate existing caches so new PDF information is immediately reflected
        self.invalidate_cache()

        logger.info(f"✅ Successfully indexed document ID {saved_doc['id']} ('{doc_title}') with {len(raw_chunks)} vector chunks.")
        return saved_doc

    def list_documents(self) -> List[Dict[str, Any]]:
        """List all indexed knowledge documents"""
        return self.knowledge_repo.list_documents()

    def delete_document(self, doc_id: int) -> bool:
        """Delete an indexed document and its chunks, flushing RAG cache"""
        deleted = self.knowledge_repo.delete_document(doc_id)
        if deleted:
            self.invalidate_cache()
        return deleted

    def query_knowledge(
        self,
        query: str,
        limit: int = 5,
        score_threshold: float = 0.15,
        doc_type: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Execute semantic RAG retrieval with 2-Tier Redis Cache (Exact Hash + Semantic Vector Matching)"""
        clean_query = query.strip()
        if not clean_query:
            return []

        scope = doc_type or "ALL"

        # Tier 1: Check Exact Hash Cache (< 0.5ms)
        exact_cached = self._get_exact_cache(clean_query, scope=scope)
        if exact_cached is not None:
            return exact_cached

        # Encode query vector
        query_vec = self.embedding_repo.encode(clean_query)

        # Tier 2: Check Semantic Vector Cache (< 5ms)
        semantic_cached = self._get_semantic_cache(
            clean_query=clean_query,
            query_vec=query_vec,
            scope=scope,
            threshold=settings.RAG_CACHE_SEMANTIC_THRESHOLD
        )
        if semantic_cached is not None:
            return semantic_cached

        # Cache Miss: Query PostgreSQL pgvector storage
        results = self.knowledge_repo.search_chunks(
            query_vector=query_vec,
            limit=limit,
            score_threshold=score_threshold,
            doc_type=doc_type
        )

        # Save to Redis Cache
        if results:
            self._save_to_cache(
                clean_query=clean_query,
                query_vec=query_vec,
                scope=scope,
                results=results
            )

        return results
