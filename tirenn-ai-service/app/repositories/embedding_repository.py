import logging
from typing import List
from sentence_transformers import SentenceTransformer
from app.core.config import settings

logger = logging.getLogger("ai-service.repository.embedding")

class EmbeddingRepository:
    """Repository responsible for computing dense vector representations using local models"""

    def __init__(self, model_name: str = settings.EMBEDDING_MODEL_NAME):
        logger.info(f"Initializing EmbeddingRepository with model: {model_name}...")
        self.model = SentenceTransformer(model_name)
        self._dim = self.model.get_embedding_dimension()
        logger.info(f"Embedding model ready with {self._dim} dimensions.")

    @property
    def dimension(self) -> int:
        return self._dim

    def encode(self, text: str) -> List[float]:
        """Encode a single string into a normalized dense float vector"""
        embedding = self.model.encode(text, normalize_embeddings=True)
        return embedding.tolist()

    def encode_batch(self, texts: List[str]) -> List[List[float]]:
        """Encode a list of strings into normalized dense float vectors in batches"""
        embeddings = self.model.encode(texts, normalize_embeddings=True, show_progress_bar=False)
        return [emb.tolist() for emb in embeddings]
