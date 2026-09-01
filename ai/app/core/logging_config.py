import time
import uuid
import logging
from contextvars import ContextVar
from starlette.middleware.base import BaseHTTPMiddleware
from fastapi import Request, Response

# Thread-safe and async task-safe context variables for Distributed Tracing
trace_id_ctx: ContextVar[str] = ContextVar("trace_id", default="-")
span_id_ctx: ContextVar[str] = ContextVar("span_id", default="-")
request_id_ctx: ContextVar[str] = ContextVar("request_id", default="-")


def get_current_trace_id() -> str:
    """Retrieve the active trace ID or '-' if outside trace scope"""
    return trace_id_ctx.get()


def get_current_span_id() -> str:
    """Retrieve the active span ID or '-' if outside trace scope"""
    return span_id_ctx.get()


def get_current_request_id() -> str:
    """Retrieve the active request ID or '-' if outside request scope"""
    return request_id_ctx.get()


def create_child_span_id() -> str:
    """Generate a new child span ID for granular operation tracing"""
    return f"span-{uuid.uuid4().hex[:8]}"


def get_tracing_headers() -> dict:
    """Generate headers dictionary with current trace/span/request IDs for downstream HTTP calls"""
    return {
        "X-Trace-ID": get_current_trace_id(),
        "X-Span-ID": create_child_span_id(),
        "X-Request-ID": get_current_request_id(),
    }


class DistributedTracingFilter(logging.Filter):
    """Logging filter that injects trace_id, span_id, and request_id into every log record"""

    def filter(self, record: logging.LogRecord) -> bool:
        record.trace_id = trace_id_ctx.get()
        record.span_id = span_id_ctx.get()
        record.request_id = request_id_ctx.get()
        return True


def setup_logging():
    """Configure unified logging with Distributed Tracing format across all loggers"""
    log_filter = DistributedTracingFilter()
    formatter = logging.Formatter(
        "%(asctime)s [%(levelname)s] [trace:%(trace_id)s] [span:%(span_id)s] [req:%(request_id)s] %(name)s: %(message)s"
    )

    root_logger = logging.getLogger()
    root_logger.setLevel(logging.INFO)

    if not root_logger.handlers:
        handler = logging.StreamHandler()
        handler.setFormatter(formatter)
        handler.addFilter(log_filter)
        root_logger.addHandler(handler)
    else:
        for handler in root_logger.handlers:
            handler.setFormatter(formatter)
            handler.addFilter(log_filter)

    for logger_name in ("uvicorn", "uvicorn.access", "uvicorn.error", "ai-service"):
        l = logging.getLogger(logger_name)
        l.addFilter(log_filter)


class DistributedTracingMiddleware(BaseHTTPMiddleware):
    """
    Middleware that extracts or generates distributed tracing headers (X-Trace-ID, X-Span-ID, X-Request-ID),
    binds them to async context variables, tracks total latency, and propagates them in HTTP response headers.
    """

    async def dispatch(self, request: Request, call_next):
        start_time = time.perf_counter()

        # 1. Read or generate Trace ID
        trace_id = (
            request.headers.get("X-Trace-ID")
            or request.headers.get("traceparent")
            or request.headers.get("X-Correlation-ID")
            or f"trace-{uuid.uuid4().hex[:16]}"
        )

        # 2. Read or generate Request ID
        req_id = (
            request.headers.get("X-Request-ID")
            or request.headers.get("X-Correlation-ID")
            or f"req-{uuid.uuid4().hex[:12]}"
        )

        # 3. Read incoming Span ID or generate new Root Span ID
        parent_span_id = request.headers.get("X-Span-ID")
        span_id = f"span-{uuid.uuid4().hex[:8]}"

        # 4. Bind to async context variables
        token_trace = trace_id_ctx.set(trace_id)
        token_span = span_id_ctx.set(span_id)
        token_req = request_id_ctx.set(req_id)

        request.state.trace_id = trace_id
        request.state.span_id = span_id
        request.state.request_id = req_id

        try:
            # 5. Process HTTP Request
            response: Response = await call_next(request)

            # 6. Inject Distributed Tracing & Latency Headers into response
            elapsed_ms = (time.perf_counter() - start_time) * 1000.0
            response.headers["X-Trace-ID"] = trace_id
            response.headers["X-Span-ID"] = span_id
            response.headers["X-Request-ID"] = req_id
            response.headers["X-Response-Time-Ms"] = f"{elapsed_ms:.2f}"
            return response
        finally:
            trace_id_ctx.reset(token_trace)
            span_id_ctx.reset(token_span)
            request_id_ctx.reset(token_req)
