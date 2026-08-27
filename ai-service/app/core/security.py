import time
import logging
from collections import defaultdict
from threading import Lock
from typing import Dict, List, Optional, Tuple
import redis
from fastapi import Request, Response, HTTPException, status
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import JSONResponse

from app.core.config import settings

logger = logging.getLogger("ai-service.security")

class SecurityHeadersMiddleware(BaseHTTPMiddleware):
    """Injects industry-standard defensive HTTP security headers"""
    async def dispatch(self, request: Request, call_next):
        response: Response = await call_next(request)
        response.headers["X-Content-Type-Options"] = "nosniff"
        response.headers["X-Frame-Options"] = "DENY"
        response.headers["X-XSS-Protection"] = "1; mode=block"
        response.headers["Referrer-Policy"] = "strict-origin-when-cross-origin"
        response.headers["Permissions-Policy"] = "camera=(), microphone=(), geolocation=()"
        return response

class RequestBodySizeLimitMiddleware(BaseHTTPMiddleware):
    """Enforces maximum request body payload limit (anti-DDoS)"""
    async def dispatch(self, request: Request, call_next):
        content_length = request.headers.get("content-length")
        if content_length:
            try:
                length = int(content_length)
                if length > settings.MAX_REQUEST_BODY_BYTES:
                    logger.warning(f"Payload size {length} exceeded max limit {settings.MAX_REQUEST_BODY_BYTES}")
                    return JSONResponse(
                        status_code=status.HTTP_413_REQUEST_ENTITY_TOO_LARGE,
                        content={
                            "success": False,
                            "error": "Payload Too Large",
                            "message": f"Request size exceeds limit of {settings.MAX_REQUEST_BODY_BYTES // (1024 * 1024)}MB."
                        }
                    )
            except ValueError:
                pass
        return await call_next(request)

class RedisSlidingWindowRateLimiter:
    """Distributed Redis-backed sliding window rate limiter with atomic Sorted Sets and in-memory fallback"""
    def __init__(self):
        self._redis_client: Optional[redis.Redis] = None
        self._memory_fallback_requests: Dict[str, List[float]] = defaultdict(list)
        self._memory_lock = Lock()
        self._init_redis()

    def _init_redis(self):
        try:
            pool = redis.ConnectionPool(
                host=settings.REDIS_HOST,
                port=settings.REDIS_PORT,
                password=settings.REDIS_PASSWORD or None,
                db=settings.REDIS_DB,
                socket_timeout=1.5,
                socket_connect_timeout=2.0,
                decode_responses=True,
                max_connections=20
            )
            self._redis_client = redis.Redis(connection_pool=pool)
            self._redis_client.ping()
            logger.info(f"🔴 Connected to Redis at {settings.REDIS_HOST}:{settings.REDIS_PORT} for rate limiting.")
        except Exception as e:
            logger.warning(f"Could not connect to Redis ({e}). Using in-memory rate limiting fallback.")
            self._redis_client = None

    def _get_client_ip(self, request: Request) -> str:
        forwarded = request.headers.get("x-forwarded-for")
        if forwarded:
            return forwarded.split(",")[0].strip()
        client = request.client
        return client.host if client else "127.0.0.1"

    def is_allowed(self, request: Request) -> Tuple[bool, int, int, int]:
        """
        Returns (is_allowed, limit, remaining_quota, retry_after_seconds)
        """
        if not settings.RATE_LIMIT_ENABLED:
            return True, 999, 999, 0

        path = request.url.path
        is_chat = "/chat" in path
        limit = settings.RATE_LIMIT_CHAT_PER_MINUTE if is_chat else settings.RATE_LIMIT_GENERAL_PER_MINUTE
        window_seconds = 60.0

        ip = self._get_client_ip(request)
        scope = "chat" if is_chat else "gen"
        redis_key = f"ratelimit:ai:{ip}:{scope}"
        now = time.time()
        cutoff = now - window_seconds

        # Attempt Redis atomic sliding window first
        if self._redis_client is not None:
            try:
                pipe = self._redis_client.pipeline()
                pipe.zremrangebyscore(redis_key, 0, cutoff)
                pipe.zcard(redis_key)
                pipe.zrange(redis_key, 0, 0, withscores=True)
                results = pipe.execute()

                current_count = results[1]
                oldest_items = results[2]

                if current_count >= limit:
                    retry_after = 60
                    if oldest_items:
                        oldest_ts = float(oldest_items[0][1])
                        retry_after = max(1, int(oldest_ts + window_seconds - now))
                    return False, limit, 0, retry_after

                add_pipe = self._redis_client.pipeline()
                member = f"{now:.6f}"
                add_pipe.zadd(redis_key, {member: now})
                add_pipe.expire(redis_key, int(window_seconds) + 5)
                add_pipe.execute()

                remaining = limit - (current_count + 1)
                return True, limit, max(0, remaining), 0

            except Exception as e:
                logger.warning(f"Redis rate limit check error ({e}). Falling back to in-memory tracking.")
                self._redis_client = None

        # In-Memory Fallback
        mem_key = f"{ip}:{scope}"
        with self._memory_lock:
            timestamps = [ts for ts in self._memory_fallback_requests[mem_key] if ts > cutoff]
            self._memory_fallback_requests[mem_key] = timestamps

            if len(timestamps) >= limit:
                oldest = timestamps[0]
                retry_after = max(1, int(oldest + window_seconds - now))
                return False, limit, 0, retry_after

            self._memory_fallback_requests[mem_key].append(now)
            remaining = limit - len(self._memory_fallback_requests[mem_key])
            return True, limit, remaining, 0

rate_limiter = RedisSlidingWindowRateLimiter()

class RateLimitMiddleware(BaseHTTPMiddleware):
    """Enforces rate limiting on incoming API traffic with standardized headers"""
    async def dispatch(self, request: Request, call_next):
        # Skip rate limit for CORS preflights, healthcheck, docs, and schema
        if request.method == "OPTIONS" or request.url.path in ["/healthz", "/docs", "/openapi.json"]:
            return await call_next(request)

        allowed, limit, remaining, retry_after = rate_limiter.is_allowed(request)
        if not allowed:
            client_ip = rate_limiter._get_client_ip(request)
            logger.warning(f"Rate limit exceeded for IP {client_ip} on {request.url.path} ({limit} req/min)")
            return JSONResponse(
                status_code=status.HTTP_429_TOO_MANY_REQUESTS,
                headers={
                    "X-RateLimit-Limit": str(limit),
                    "X-RateLimit-Remaining": "0",
                    "X-RateLimit-Reset": str(retry_after),
                    "Retry-After": str(retry_after),
                },
                content={
                    "success": False,
                    "error": f"limit_{limit}_exceeded",
                    "message": f"Rate limit exceeded ({limit} req/60s). Please wait {retry_after} seconds.",
                    "retry_after": retry_after
                }
            )

        response: Response = await call_next(request)
        response.headers["X-RateLimit-Limit"] = str(limit)
        response.headers["X-RateLimit-Remaining"] = str(remaining)
        response.headers["X-RateLimit-Reset"] = "60"
        return response

def verify_internal_api_key(request: Request):
    """Optional dependency to secure admin sync/indexing endpoints with an API key"""
    if not settings.INTERNAL_API_KEY:
        return True

    auth_header = request.headers.get("x-api-key") or request.headers.get("authorization", "")
    if auth_header.startswith("Bearer "):
        auth_header = auth_header[7:]

    if auth_header != settings.INTERNAL_API_KEY:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid or missing X-API-Key / Authorization header for internal endpoint"
        )
    return True
