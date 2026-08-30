import json
import logging
from typing import List, Optional
import redis
from app.core.config import settings
from app.domain.chat import ChatMessage

logger = logging.getLogger("ai-service.repository.session")

class SessionRepository:
    """Manages conversational session history in Redis Lists with sliding window bounding and TTL expiration"""

    def __init__(
        self,
        ttl_seconds: Optional[int] = None,
        history_limit: Optional[int] = None,
        max_stored: Optional[int] = None
    ):
        self.ttl_seconds = ttl_seconds or settings.SESSION_TTL_SECONDS
        self.history_limit = history_limit or settings.SESSION_HISTORY_LIMIT
        self.max_stored = max_stored or settings.SESSION_MAX_STORED
        self._client: Optional[redis.Redis] = None
        self._init_client()

    def _init_client(self):
        try:
            self._client = redis.Redis(
                host=settings.REDIS_HOST,
                port=settings.REDIS_PORT,
                password=settings.REDIS_PASSWORD if settings.REDIS_PASSWORD else None,
                db=settings.REDIS_DB,
                decode_responses=True,
                socket_timeout=2.0,
                socket_connect_timeout=2.0
            )
            self._client.ping()
            logger.info(f"✅ [REDIS] Connected to Redis session store at {settings.REDIS_HOST}:{settings.REDIS_PORT}")
        except Exception as e:
            logger.warning(f"⚠️ [REDIS] Failed to connect to Redis ({e}). Session persistence will be disabled in fallback mode.")
            self._client = None

    def _get_key(self, session_id: str) -> str:
        return f"chat:session:{session_id}"

    def get_history(self, session_id: str, limit: Optional[int] = None) -> List[ChatMessage]:
        """Fetch the last N messages for a given session ID from Redis List (sliding window buffer)"""
        if not self._client or not session_id:
            return []
        try:
            key = self._get_key(session_id)
            n = limit if (limit and limit > 0) else self.history_limit
            # -n to -1 retrieves the tail of the list (oldest to newest among the last N)
            raw_list = self._client.lrange(key, -n, -1)
            if not raw_list:
                return []
            results: List[ChatMessage] = []
            for raw in raw_list:
                try:
                    m = json.loads(raw)
                    results.append(ChatMessage(role=m["role"], content=m["content"]))
                except Exception:
                    continue
            return results
        except Exception as e:
            logger.warning(f"Error fetching session history from Redis list for {session_id}: {e}")
            return []

    def append_messages(self, session_id: str, messages: List[ChatMessage]) -> bool:
        """Atomically append new chat messages to the Redis List, trim older history, and refresh TTL"""
        if not self._client or not session_id or not messages:
            return False
        try:
            key = self._get_key(session_id)
            pipe = self._client.pipeline()
            for m in messages:
                role = m.role if hasattr(m, 'role') else m.get('role', 'user')
                content = m.content if hasattr(m, 'content') else m.get('content', '')
                payload = json.dumps({"role": role, "content": content}, ensure_ascii=False, default=str)
                pipe.rpush(key, payload)
            if self.max_stored > 0:
                pipe.ltrim(key, -self.max_stored, -1)
            pipe.expire(key, self.ttl_seconds)
            pipe.execute()
            return True
        except Exception as e:
            logger.warning(f"Error appending messages to Redis session {session_id}: {e}")
            return False

    def append_message(self, session_id: str, message: ChatMessage) -> bool:
        """Atomically append a single message to the Redis List"""
        return self.append_messages(session_id, [message])

    def save_history(self, session_id: str, messages: List[ChatMessage]) -> bool:
        """Overwrite/initialize session history with provided messages (backward-compatible)"""
        if not self._client or not session_id:
            return False
        try:
            key = self._get_key(session_id)
            pipe = self._client.pipeline()
            pipe.delete(key)
            for m in messages:
                payload = json.dumps({"role": m.role, "content": m.content}, ensure_ascii=False)
                pipe.rpush(key, payload)
            if self.max_stored > 0:
                pipe.ltrim(key, -self.max_stored, -1)
            pipe.expire(key, self.ttl_seconds)
            pipe.execute()
            return True
        except Exception as e:
            logger.warning(f"Error saving session history to Redis for {session_id}: {e}")
            return False

    def delete_session(self, session_id: str) -> bool:
        """Explicitly delete a session's history from Redis"""
        if not self._client or not session_id:
            return False
        try:
            key = self._get_key(session_id)
            deleted = self._client.delete(key)
            logger.info(f"🗑️ [REDIS] Session deleted: {key}")
            return deleted > 0
        except Exception as e:
            logger.warning(f"Error deleting session {session_id} from Redis: {e}")
            return False
