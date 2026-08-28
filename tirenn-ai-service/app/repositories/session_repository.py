import json
import logging
from typing import List, Optional
import redis
from app.core.config import settings
from app.domain.chat import ChatMessage

logger = logging.getLogger("ai-service.repository.session")

class SessionRepository:
    """Manages conversational session history in Redis with TTL expiration and on-demand deletion"""

    def __init__(self, ttl_seconds: int = 86400):
        self.ttl_seconds = ttl_seconds
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

    def get_history(self, session_id: str) -> List[ChatMessage]:
        """Fetch message history for a given session ID from Redis"""
        if not self._client or not session_id:
            return []
        try:
            key = self._get_key(session_id)
            raw = self._client.get(key)
            if not raw:
                return []
            data = json.loads(raw)
            return [ChatMessage(role=m["role"], content=m["content"]) for m in data]
        except Exception as e:
            logger.warning(f"Error fetching session history from Redis for {session_id}: {e}")
            return []

    def save_history(self, session_id: str, messages: List[ChatMessage]) -> bool:
        """Save/overwrite full message history for a session ID with TTL (auto-delete when not in use)"""
        if not self._client or not session_id:
            return False
        try:
            key = self._get_key(session_id)
            data = [{"role": m.role, "content": m.content} for m in messages]
            self._client.setex(key, self.ttl_seconds, json.dumps(data, ensure_ascii=False))
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
