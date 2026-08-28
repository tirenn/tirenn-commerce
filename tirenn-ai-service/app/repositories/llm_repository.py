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
        timeout: float = 60.0
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
            if resp.status_code == 404:
                # If model was missing, attempt auto-pull and retry once
                logger.warning(f"Ollama returned 404 for model '{self.model_name}'. Attempting auto-pull...")
                await self.ensure_model_available()
                resp = await client.post(f"{self.base_url}/api/chat", json=payload)

            resp.raise_for_status()
            data = resp.json()
            return data.get("message", {})
