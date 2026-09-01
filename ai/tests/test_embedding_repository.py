import pytest
from unittest.mock import MagicMock, patch
import httpx
import numpy as np

from app.repositories.embedding_repository import EmbeddingRepository


def test_embedding_repository_init_and_dimension():
    """Verify EmbeddingRepository initializes with correct base url, model name, and dimension"""
    mock_resp = MagicMock()
    mock_resp.status_code = 200
    mock_resp.json.return_value = {"embeddings": [[0.1] * 1024]}

    with patch("httpx.Client.post", return_value=mock_resp):
        repo = EmbeddingRepository(base_url="http://localhost:11434", model_name="bge-m3")
        assert repo.dimension == 1024
        assert repo.base_url == "http://localhost:11434"
        assert repo.model_name == "bge-m3"
        repo.close()


def test_embedding_repository_encode_empty():
    """Verify encode returns zero vector when given empty text"""
    repo = EmbeddingRepository(base_url="http://localhost:11434", model_name="bge-m3")
    repo._dim = 384
    vec = repo.encode("")
    assert len(vec) == 384
    assert all(v == 0.0 for v in vec)

    vec_space = repo.encode("   ")
    assert len(vec_space) == 384
    assert all(v == 0.0 for v in vec_space)
    repo.close()


def test_embedding_repository_encode_single_normalized():
    """Verify encode calls Ollama and normalizes result to unit length"""
    raw_vector = [3.0, 4.0] + [0.0] * 382  # L2 norm is 5.0
    mock_resp = MagicMock()
    mock_resp.status_code = 200
    mock_resp.json.return_value = {"embeddings": [raw_vector]}

    with patch("httpx.Client.post", return_value=mock_resp):
        repo = EmbeddingRepository(base_url="http://localhost:11434", model_name="bge-m3")
        vec = repo.encode("sepatu lari nike")
        assert len(vec) == 384
        # Normalized [3.0/5.0, 4.0/5.0, ...] -> [0.6, 0.8, ...]
        assert pytest.approx(vec[0], 0.001) == 0.6
        assert pytest.approx(vec[1], 0.001) == 0.8
        # Unit norm check
        norm = np.linalg.norm(np.array(vec))
        assert pytest.approx(norm, 0.001) == 1.0
        repo.close()


def test_embedding_repository_encode_batch():
    """Verify encode_batch calls Ollama /api/embed batch endpoint and normalizes vectors"""
    raw_vectors = [
        [1.0] * 384,
        [2.0] * 384,
    ]
    mock_resp = MagicMock()
    mock_resp.status_code = 200
    mock_resp.json.return_value = {"embeddings": raw_vectors}

    with patch("httpx.Client.post", return_value=mock_resp):
        repo = EmbeddingRepository(base_url="http://localhost:11434", model_name="bge-m3")
        results = repo.encode_batch(["baju batik", "celana jeans"])
        assert len(results) == 2
        assert len(results[0]) == 384
        assert len(results[1]) == 384
        for r in results:
            assert pytest.approx(np.linalg.norm(np.array(r)), 0.001) == 1.0
        repo.close()


def test_embedding_repository_encode_batch_empty():
    """Verify encode_batch returns empty list when input is empty"""
    repo = EmbeddingRepository(base_url="http://localhost:11434", model_name="bge-m3")
    results = repo.encode_batch([])
    assert results == []
    repo.close()


def test_embedding_repository_fallback_on_error():
    """Verify fallback to deterministic vector if Ollama is unreachable"""
    with patch("httpx.Client.post", side_effect=httpx.ConnectError("Connection refused")):
        repo = EmbeddingRepository(base_url="http://unreachable:11434", model_name="bge-m3")
        vec = repo.encode("fallback test query")
        assert len(vec) == repo.dimension
        assert pytest.approx(np.linalg.norm(np.array(vec)), 0.001) == 1.0
        repo.close()
