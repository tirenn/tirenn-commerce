import logging
from typing import List, Optional
import httpx
import numpy as np

from app.core.config import settings

logger = logging.getLogger("ai-service.repository.embedding")


class EmbeddingRepository:
    """Repository responsible for computing dense vector representations using Ollama embedding API"""

    def __init__(
        self,
        base_url: str = settings.OLLAMA_BASE_URL,
        model_name: str = settings.EMBEDDING_MODEL_NAME,
        timeout: float = 60.0
    ):
        self.base_url = base_url.rstrip("/")
        self.model_name = model_name
        self.timeout = timeout
        self._dim = 384
        logger.info(f"Initializing EmbeddingRepository with Ollama at {self.base_url} (model: {self.model_name})...")
        self._client = httpx.Client(
            timeout=httpx.Timeout(self.timeout, connect=10.0, read=self.timeout, write=10.0)
        )
        self._detect_dimension()

    def _detect_dimension(self):
        """Attempt a sample embedding to discover vector dimension from Ollama"""
        try:
            vec = self._embed_single("test")
            if vec and len(vec) > 0:
                self._dim = len(vec)
                logger.info(f"Ollama embedding model '{self.model_name}' ready with {self._dim} dimensions.")
        except Exception as e:
            logger.warning(
                f"Could not connect to Ollama embedding service at init ({e}). Defaulting to {self._dim} dimensions."
            )

    @property
    def dimension(self) -> int:
        return self._dim

    def _normalize(self, vec: List[float]) -> List[float]:
        """Normalize a float vector to unit length (L2 norm)"""
        arr = np.array(vec, dtype=np.float32)
        norm = np.linalg.norm(arr)
        if norm > 0:
            arr = arr / norm
        return arr.tolist()

    def _embed_single(self, text: str) -> List[float]:
        """Internal helper to call Ollama /api/embed or /api/embeddings for a single text"""
        # Try /api/embed first (Ollama modern endpoint)
        try:
            resp = self._client.post(
                f"{self.base_url}/api/embed",
                json={"model": self.model_name, "input": text}
            )
            if resp.status_code == 200:
                data = resp.json()
                embeddings = data.get("embeddings")
                if embeddings and len(embeddings) > 0:
                    return embeddings[0]
        except Exception:
            pass

        # Fallback to standard /api/embeddings
        resp = self._client.post(
            f"{self.base_url}/api/embeddings",
            json={"model": self.model_name, "prompt": text}
        )
        resp.raise_for_status()
        data = resp.json()
        return data.get("embedding", [])

    def _embed_batch(self, texts: List[str]) -> List[List[float]]:
        """Internal helper to call Ollama /api/embed for multiple texts"""
        try:
            resp = self._client.post(
                f"{self.base_url}/api/embed",
                json={"model": self.model_name, "input": texts}
            )
            if resp.status_code == 200:
                data = resp.json()
                embeddings = data.get("embeddings")
                if embeddings and len(embeddings) == len(texts):
                    return embeddings
        except Exception as e:
            logger.debug(f"/api/embed batch call failed ({e}), falling back to iterative single embedding")

        # Fallback to sequential single embedding
        return [self._embed_single(t) for t in texts]

    def encode(self, text: str) -> List[float]:
        """Encode a single string into a normalized dense float vector via Ollama"""
        if not text or not text.strip():
            return [0.0] * self._dim

        try:
            raw_vec = self._embed_single(text)
            if raw_vec and len(raw_vec) > 0:
                self._dim = len(raw_vec)
                return self._normalize(raw_vec)
        except Exception as e:
            logger.warning(
                f"Ollama embedding error for text '{text[:30]}...': {e}. Falling back to deterministic vector."
            )

        import hashlib
        h = hashlib.sha256(text.encode("utf-8")).digest()
        vec = [(b / 255.0) for b in (h * 12)[:self._dim]]
        return self._normalize(vec)

    def encode_batch(self, texts: List[str]) -> List[List[float]]:
        """Encode a list of strings into normalized dense float vectors in batches via Ollama"""
        if not texts:
            return []

        try:
            raw_vectors = self._embed_batch(texts)
            if raw_vectors and len(raw_vectors) == len(texts):
                if raw_vectors[0]:
                    self._dim = len(raw_vectors[0])
                return [self._normalize(v) if v else [0.0] * self._dim for v in raw_vectors]
        except Exception as e:
            logger.warning(f"Ollama batch embedding error: {e}. Falling back to single encode.")

        return [self.encode(t) for t in texts]

    def close(self):
        """Close the underlying HTTP client"""
        if hasattr(self, "_client") and self._client:
            self._client.close()
