import logging
from typing import List, Dict, Any, Optional
import httpx
from app.core.config import settings

logger = logging.getLogger("ai-service.repository.llm")

class LLMRepository:
    """Repository handling communication with local Ollama LLM service"""

    def __init__(self, base_url: str = settings.OLLAMA_BASE_URL, model_name: str = settings.LLM_MODEL):
        self.base_url = base_url.rstrip("/")
        self.model_name = model_name
        logger.info(f"LLMRepository configured with Ollama at {self.base_url} (model: {self.model_name})")

    async def chat(
        self,
        messages: List[Dict[str, Any]],
        tools: Optional[List[Dict[str, Any]]] = None,
        temperature: float = 0.0,
        timeout: float = 45.0
    ) -> Dict[str, Any]:
        """Send chat messages and tool definitions to Ollama and return assistant message"""
        payload: Dict[str, Any] = {
            "model": self.model_name,
            "messages": messages,
            "stream": False,
            "options": {
                "temperature": temperature
            }
        }
        if tools:
            payload["tools"] = tools

        async with httpx.AsyncClient(timeout=timeout) as client:
            resp = await client.post(f"{self.base_url}/api/chat", json=payload)
            resp.raise_for_status()
            data = resp.json()
            return data.get("message", {})
