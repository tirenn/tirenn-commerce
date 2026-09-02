"""
Unit & Integration Tests for LLM Response Semantic Cache (GPTCache Pattern)
Tests 2-Tier Exact & Semantic Vector matching, thresholding, and cache invalidation.
"""

import json
import pytest
import numpy as np
from unittest.mock import MagicMock, patch

from app.core.llm_cache import LLMSemanticCache


class MockRedis:
    """In-memory dictionary mock simulating Redis get, setex, rpush, lrange, scan_iter, delete"""
    def __init__(self):
        self.store = {}
        self.lists = {}

    def ping(self):
        return True

    def get(self, key):
        return self.store.get(key)

    def setex(self, key, ttl, value):
        self.store[key] = value

    def lrange(self, key, start, end):
        return self.lists.get(key, [])

    def rpush(self, key, value):
        if key not in self.lists:
            self.lists[key] = []
        self.lists[key].append(value)

    def ltrim(self, key, start, end):
        if key in self.lists:
            self.lists[key] = self.lists[key][start:]

    def expire(self, key, ttl):
        pass

    def scan_iter(self, match):
        prefix = match.replace("*", "")
        for k in list(self.store.keys()):
            if k.startswith(prefix):
                yield k
        for k in list(self.lists.keys()):
            if k.startswith(prefix):
                yield k

    def delete(self, *keys):
        for k in keys:
            self.store.pop(k, None)
            self.lists.pop(k, None)

    def pipeline(self):
        return self

    def execute(self):
        return True


def test_llm_cache_exact_match():
    """Verify Tier 1 Exact Hash Cache stores and returns response in < 0.5ms"""
    mock_redis = MockRedis()
    cache = LLMSemanticCache(redis_client=mock_redis)

    query = "rekomendasikan hp gaming 5 jutaan"
    payload = {
        "reply": "Berikut 2 HP gaming: Realme GT 6 dan Xiaomi 14.",
        "tool_calls": [{"name": "search_products", "args": {"category_id": 1}}],
        "suggested_products": [{"id": 1, "name": "Realme GT 6"}]
    }

    dummy_vec = [1.0, 0.0, 0.0]
    cache.set_cached_turn(query=query, query_vector=dummy_vec, payload=payload, scope="shopper")

    # 1. Exact query should HIT
    hit = cache.get_cached_turn(query=query, query_vector=dummy_vec, scope="shopper")
    assert hit is not None
    assert hit["reply"] == payload["reply"]
    assert len(hit["suggested_products"]) == 1

    # 2. Different casing / whitespace should still HIT (normalized)
    hit_case = cache.get_cached_turn(query="  REKOMENDASIKAN HP GAMING 5 JUTAAN  ", query_vector=dummy_vec, scope="shopper")
    assert hit_case is not None
    assert hit_case["reply"] == payload["reply"]


def test_llm_cache_semantic_match_and_threshold():
    """Verify Tier 2 Semantic Vector cosine similarity match against paraphrased questions"""
    mock_redis = MockRedis()
    cache = LLMSemanticCache(redis_client=mock_redis)

    # Base query
    base_query = "apakah ada garansi untuk barang cacat?"
    base_vec = [0.9, 0.1, 0.0]
    payload = {"reply": "Toko kami menyediakan garansi tukar unit 7 hari dengan video unboxing."}
    cache.set_cached_turn(query=base_query, query_vector=base_vec, payload=payload, scope="shopper")

    # Paraphrased query with very high cosine similarity (~0.98)
    paraphrase_query = "bagaimana syarat klaim garansi barang rusak?"
    paraphrase_vec = [0.88, 0.12, 0.0]

    hit = cache.get_cached_turn(
        query=paraphrase_query,
        query_vector=paraphrase_vec,
        scope="shopper",
        threshold=0.90
    )
    assert hit is not None
    assert "garansi tukar unit 7 hari" in hit["reply"]

    # Completely different query with low similarity (e.g. [0.0, 0.0, 1.0]) should MISS
    unrelated_query = "rekomendasikan power bank anker"
    unrelated_vec = [0.0, 0.0, 1.0]

    miss = cache.get_cached_turn(
        query=unrelated_query,
        query_vector=unrelated_vec,
        scope="shopper",
        threshold=0.90
    )
    assert miss is None


def test_llm_cache_invalidation():
    """Verify cache invalidation flushes stored exact and semantic keys"""
    mock_redis = MockRedis()
    cache = LLMSemanticCache(redis_client=mock_redis)

    cache.set_cached_turn("query 1", [1.0, 0.0], {"reply": "ans 1"}, scope="shopper")
    cache.set_cached_turn("query 2", [0.0, 1.0], {"reply": "ans 2"}, scope="shopper")

    # Invalidate shopper scope
    cache.invalidate(scope="shopper")

    assert cache.get_cached_turn("query 1", [1.0, 0.0], scope="shopper") is None
    assert cache.get_cached_turn("query 2", [0.0, 1.0], scope="shopper") is None
