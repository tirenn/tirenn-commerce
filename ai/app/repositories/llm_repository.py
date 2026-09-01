import time
import json
import logging
from typing import List, Dict, Any, Optional
import httpx
from app.core.config import settings
from app.core.logging_config import get_tracing_headers

logger = logging.getLogger("ai-service.repository.llm")

class LLMRepository:
    """Repository handling communication with local Ollama LLM service"""

    def __init__(self, base_url: str = settings.OLLAMA_BASE_URL, model_name: str = settings.LLM_MODEL):
        self.base_url = base_url.rstrip("/")
        self.model_name = model_name
        logger.info(f"LLMRepository configured with Ollama at {self.base_url} (model: {self.model_name})")

    async def ensure_model_available(self):
        """Check if model exists in Ollama, if not trigger background pull"""
        try:
            async with httpx.AsyncClient(timeout=10.0) as client:
                res = await client.get(f"{self.base_url}/api/tags")
                if res.status_code == 200:
                    models = [m.get("name", "") for m in res.json().get("models", [])]
                    if not any(self.model_name in m for m in models):
                        logger.warning(f"⚠️ Model '{self.model_name}' not found in Ollama tags ({models}). Triggering pull...")
                        pull_res = await client.post(
                            f"{self.base_url}/api/pull",
                            json={"name": self.model_name, "stream": False},
                            timeout=300.0
                        )
                        if pull_res.status_code == 200:
                            logger.info(f"✅ Model '{self.model_name}' pulled successfully into Ollama.")
                        else:
                            logger.error(f"❌ Failed to auto-pull model: {pull_res.text}")
                    else:
                        logger.info(f"✅ Verified model '{self.model_name}' is available in Ollama.")
        except Exception as e:
            logger.warning(f"Could not verify/pull Ollama model automatically: {e}")

    async def chat(
        self,
        messages: List[Dict[str, Any]],
        tools: Optional[List[Dict[str, Any]]] = None,
        temperature: float = 0.0,
        timeout: Optional[float] = None
    ) -> Dict[str, Any]:
        """Send chat messages and tool definitions to Ollama, log model input & response, and return assistant message"""
        effective_timeout = timeout if timeout is not None else settings.LLM_TIMEOUT
        tool_names = [t.get("function", {}).get("name") for t in (tools or [])]

        # 1. Log Input to Model
        latest_msg = messages[-1] if messages else {}
        logger.info(
            f"🚀 [LLM_INPUT] Model: '{self.model_name}' | Temp: {temperature} | "
            f"MsgCount: {len(messages)} | ToolsAvailable: {tool_names} | "
            f"LatestTurn: {latest_msg.get('role')} -> {str(latest_msg.get('content'))[:180]}"
        )

        payload: Dict[str, Any] = {
            "model": self.model_name,
            "messages": messages,
            "stream": False,
            "keep_alive": settings.LLM_KEEP_ALIVE,
            "options": {
                "temperature": temperature,
                "num_predict": settings.LLM_NUM_PREDICT,
                "num_ctx": settings.LLM_NUM_CTX
            }
        }
        if tools:
            payload["tools"] = tools

        start_time = time.perf_counter()
        client_timeout = httpx.Timeout(effective_timeout, connect=15.0, read=effective_timeout, write=15.0)
        tracing_headers = get_tracing_headers()

        async with httpx.AsyncClient(timeout=client_timeout) as client:
            resp = await client.post(f"{self.base_url}/api/chat", json=payload, headers=tracing_headers)
            if resp.status_code == 404:
                logger.warning(f"Ollama returned 404 for model '{self.model_name}'. Attempting auto-pull...")
                await self.ensure_model_available()
                resp = await client.post(f"{self.base_url}/api/chat", json=payload, headers=tracing_headers)

            resp.raise_for_status()
            data = resp.json()

        duration_ms = (time.perf_counter() - start_time) * 1000.0
        assistant_msg = data.get("message", {})
        tool_calls = assistant_msg.get("tool_calls", [])
        content = assistant_msg.get("content", "")

        # 2. Log Response from Model
        tool_call_summaries = [
            f"{tc.get('function', {}).get('name')}({json.dumps(tc.get('function', {}).get('arguments', {}), ensure_ascii=False, default=str)})"
            for tc in tool_calls
        ]
        logger.info(
            f"🤖 [LLM_RESPONSE] Duration: {duration_ms:.2f}ms | "
            f"ToolCallsCount: {len(tool_calls)} | "
            f"Calls: {tool_call_summaries if tool_calls else 'None'} | "
            f"ContentPreview: {content[:200] if content else '(tool_calling_turn)'}"
        )

        return assistant_msg
