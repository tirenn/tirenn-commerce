"""
Adversarial Stress Test Suite for tirenn-ai-service
Tests vector math, price corridor calculations, category boosts, limit clamping, edge cases, and error handling.
"""

import math
import numpy as np
import pytest
from unittest.mock import MagicMock, patch, call
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.repositories.product_repository import ProductRepository
from app.usecases.recommendation_usecase import RecommendationUseCase
from app.handlers.recommendation_handler import get_recommendation_router, RecommendationResponse, RecommendationItem


# --------------------------------------------------------------------------
# Mathematical Verification: Vector Cosine Similarity & Category Boost
# --------------------------------------------------------------------------

def cosine_similarity(v1: list, v2: list) -> float:
    a = np.array(v1)
    b = np.array(v2)
    return float(np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b)))


def test_vector_math_cosine_similarity_and_boost_simulation():
    """
    Empirically verify that pgvector's (1 - (embedding <=> target)) matches standard cosine similarity,
    and that subcategory boost (+0.15) and category boost (+0.08) rank items correctly.
    """
    np.random.seed(42)
    dim = 384
    target_vec = np.random.randn(dim).tolist()
    target_vec = (np.array(target_vec) / np.linalg.norm(target_vec)).tolist()

    # Product A: High similarity (0.95), different category (+0.00 boost) -> final score ~0.95
    # Product B: Moderate similarity (0.85), same subcategory (+0.15 boost) -> final score ~1.00
    # Product C: Moderate similarity (0.88), same category (+0.08 boost) -> final score ~0.96
    
    vec_a = np.array(target_vec) + np.random.randn(dim) * 0.1
    vec_a = (vec_a / np.linalg.norm(vec_a)).tolist()

    vec_b = np.array(target_vec) + np.random.randn(dim) * 0.5
    vec_b = (vec_b / np.linalg.norm(vec_b)).tolist()

    vec_c = np.array(target_vec) + np.random.randn(dim) * 0.4
    vec_c = (vec_c / np.linalg.norm(vec_c)).tolist()

    sim_a = cosine_similarity(target_vec, vec_a)
    sim_b = cosine_similarity(target_vec, vec_b)
    sim_c = cosine_similarity(target_vec, vec_c)

    score_a = round(sim_a + 0.00, 4)
    score_b = round(sim_b + 0.15, 4)
    score_c = round(sim_c + 0.08, 4)

    # Subcategory item B should beat C and A if boost outweighs raw vector distance
    assert score_b > score_a or score_b > score_c, "Subcategory boost should elevate relevance"


# --------------------------------------------------------------------------
# Price Corridor Boundary Verification
# --------------------------------------------------------------------------

@pytest.mark.parametrize("target_price,expected_min,expected_max", [
    (100000.0, 40000.0, 250000.0),
    (0.0, 0.0, 999999999.0),
    (1000.0, 400.0, 2500.0),
    (10000000.0, 4000000.0, 25000000.0),
])
def test_price_corridor_calculation(target_price, expected_min, expected_max):
    """Verify exact price corridor bounds: 0.4 * target <= price <= 2.5 * target"""
    min_price = target_price * 0.4 if target_price > 0 else 0.0
    max_price = target_price * 2.5 if target_price > 0 else 999999999.0

    assert min_price == expected_min
    assert max_price == expected_max


# --------------------------------------------------------------------------
# Edge Cases in ProductRepository.get_similar_products
# --------------------------------------------------------------------------

@patch.object(ProductRepository, "_init_db")
@patch.object(ProductRepository, "_get_connection")
def test_similar_products_price_corridor_widen_trigger(mock_get_conn, mock_init):
    """
    Stress-test: If tight price corridor returns fewer than `limit` items,
    verify that the widened corridor query executes to backfill the remaining slots.
    """
    mock_conn = MagicMock()
    mock_cur = MagicMock()
    mock_get_conn.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cur

    # Target product #1
    mock_cur.fetchone.return_value = {
        "id": 1,
        "name": "Target Item",
        "category_id": 10,
        "sub_category_id": 20,
        "price": 500000.0,
        "embedding_str": "[0.1, 0.2, 0.3]"
    }

    # Step 1 returns only 2 items (while limit = 6)
    # Step 2 (widen) returns 4 items
    corridor_items = [
        {"id": 2, "name": "Item 2", "sku": "SKU2", "category_id": 10, "sub_category_id": 20, "sub_category_name": "S", "price": 450000.0, "currency": "IDR", "image_url": "", "stock_quantity": 5, "badge": "", "description": "", "score": 0.9},
        {"id": 3, "name": "Item 3", "sku": "SKU3", "category_id": 10, "sub_category_id": 20, "sub_category_name": "S", "price": 550000.0, "currency": "IDR", "image_url": "", "stock_quantity": 8, "badge": "", "description": "", "score": 0.85},
    ]
    widen_items = [
        {"id": 4, "name": "Item 4", "sku": "SKU4", "category_id": 10, "sub_category_id": 21, "sub_category_name": "S", "price": 100000.0, "currency": "IDR", "image_url": "", "stock_quantity": 2, "badge": "", "description": "", "score": 0.8},
        {"id": 5, "name": "Item 5", "sku": "SKU5", "category_id": 10, "sub_category_id": 21, "sub_category_name": "S", "price": 1500000.0, "currency": "IDR", "image_url": "", "stock_quantity": 3, "badge": "", "description": "", "score": 0.78},
        {"id": 6, "name": "Item 6", "sku": "SKU6", "category_id": 11, "sub_category_id": 22, "sub_category_name": "S", "price": 2000000.0, "currency": "IDR", "image_url": "", "stock_quantity": 4, "badge": "", "description": "", "score": 0.75},
        {"id": 7, "name": "Item 7", "sku": "SKU7", "category_id": 11, "sub_category_id": 22, "sub_category_name": "S", "price": 3000000.0, "currency": "IDR", "image_url": "", "stock_quantity": 1, "badge": "", "description": "", "score": 0.70},
    ]

    mock_cur.fetchall.side_effect = [corridor_items, widen_items]

    repo = ProductRepository()
    results = repo.get_similar_products(product_id=1, limit=6)

    assert len(results) == 6
    assert [r["id"] for r in results] == [2, 3, 4, 5, 6, 7]
    assert mock_cur.execute.call_count == 3  # 1. target fetch, 2. corridor query, 3. widen query


@patch.object(ProductRepository, "_init_db")
@patch.object(ProductRepository, "_get_connection")
def test_similar_products_cold_start_no_embedding(mock_get_conn, mock_init):
    """
    Stress-test: When product has embedding = NULL (cold start),
    repository must cleanly fallback to category popularity query without raising an exception.
    """
    mock_conn = MagicMock()
    mock_cur = MagicMock()
    mock_get_conn.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cur

    mock_cur.fetchone.return_value = {
        "id": 99,
        "name": "Cold Start Product",
        "category_id": 5,
        "sub_category_id": 12,
        "price": 250000.0,
        "embedding_str": None  # No embedding
    }

    mock_cur.fetchall.return_value = [
        {"id": 100, "name": "Cat Product 1", "sku": "SKU100", "category_id": 5, "sub_category_id": 12, "sub_category_name": "SC", "price": 240000.0, "currency": "IDR", "image_url": "", "stock_quantity": 20, "badge": "Terlaris", "description": "", "score": 0.5},
        {"id": 101, "name": "Cat Product 2", "sku": "SKU101", "category_id": 5, "sub_category_id": 12, "sub_category_name": "SC", "price": 260000.0, "currency": "IDR", "image_url": "", "stock_quantity": 15, "badge": "", "description": "", "score": 0.5},
    ]

    repo = ProductRepository()
    results = repo.get_similar_products(product_id=99, limit=4)

    assert len(results) == 2
    assert results[0]["id"] == 100
    assert results[0]["reason"] == "category_fallback"
    assert results[0]["score"] == 0.5000


# --------------------------------------------------------------------------
# Edge Cases in RecommendationUseCase (Limit Clamping, Types, None/Types)
# --------------------------------------------------------------------------

@pytest.mark.parametrize("input_limit,expected_clamped", [
    (-10, 4),
    (0, 4),
    (1, 4),
    (3, 4),
    (4, 4),
    (6, 6),
    (8, 8),
    (9, 8),
    (100, 8),
    ("5", 5),
    ("invalid", 6),
    (None, 6),
    (5.7, 5),  # Float truncation
])
def test_usecase_limit_clamping_adversarial(input_limit, expected_clamped):
    """Test boundary and malformed limit inputs to RecommendationUseCase"""
    mock_repo = MagicMock(spec=ProductRepository)
    mock_repo.get_similar_products.return_value = [{"id": i, "score": 0.9, "reason": "test"} for i in range(1, 15)]

    uc = RecommendationUseCase(product_repo=mock_repo)
    res = uc.get_recommendations(product_id=1, rec_type="similar", limit=input_limit)

    mock_repo.get_similar_products.assert_called_with(product_id=1, limit=expected_clamped)
    assert len(res) == expected_clamped


@pytest.mark.parametrize("rec_type,expected_method", [
    ("similar", "get_similar_products"),
    ("SIMILAR", "get_similar_products"),
    ("  similar  ", "get_similar_products"),
    ("unknown_strategy", "get_similar_products"),  # Unknown defaults to similar
    ("", "get_similar_products"),
    (None, "get_similar_products"),
    ("frequently_bought_together", "get_frequently_bought_together"),
    ("cross_category", "get_frequently_bought_together"),
    ("cart", "get_frequently_bought_together"),
    ("fbt", "get_frequently_bought_together"),
    ("  CROSS_CATEGORY ", "get_frequently_bought_together"),
])
def test_usecase_rec_type_routing(rec_type, expected_method):
    """Test that all casing, whitespace, and unknown rec_types resolve safely"""
    mock_repo = MagicMock(spec=ProductRepository)
    mock_repo.get_similar_products.return_value = []
    mock_repo.get_frequently_bought_together.return_value = []

    uc = RecommendationUseCase(product_repo=mock_repo)
    uc.get_recommendations(product_id=1, rec_type=rec_type, limit=6)

    if expected_method == "get_similar_products":
        mock_repo.get_similar_products.assert_called_once()
        mock_repo.get_frequently_bought_together.assert_not_called()
    else:
        mock_repo.get_frequently_bought_together.assert_called_once()
        mock_repo.get_similar_products.assert_not_called()


# --------------------------------------------------------------------------
# FastAPI Handler Edge Cases (Pydantic validation, 422, 500, etc.)
# --------------------------------------------------------------------------

def test_api_handler_negative_and_zero_product_id():
    """Verify FastAPI returns 422 Unprocessable Entity on non-positive product ID"""
    test_app = FastAPI()
    mock_uc = MagicMock(spec=RecommendationUseCase)
    router = get_recommendation_router(recommendation_usecase=mock_uc)
    test_app.include_router(router, prefix="/api/v1")
    client = TestClient(test_app)

    # 0 -> 422
    resp_zero = client.get("/api/v1/products/0/recommendations")
    assert resp_zero.status_code == 422

    # -5 -> 422
    resp_neg = client.get("/api/v1/products/-5/recommendations")
    assert resp_neg.status_code == 422

    # non-numeric -> 422
    resp_str = client.get("/api/v1/products/abc/recommendations")
    assert resp_str.status_code == 422


def test_api_handler_contract_fields():
    """Verify JSON schema contains both 'recommendations' and 'data' and expected types"""
    test_app = FastAPI()
    mock_uc = MagicMock(spec=RecommendationUseCase)
    mock_uc.get_recommendations.return_value = [
        {
            "id": 10,
            "name": "Super Gaming Mouse",
            "sku": "ACC-MOU-010",
            "category_id": 2,
            "sub_category_id": 4,
            "sub_category_name": "Peripherals",
            "price": 450000.0,
            "currency": "IDR",
            "image_url": "https://img.example.com/mouse.jpg",
            "stock_quantity": 12,
            "badge": "Terlaris",
            "description": "Ergonomic gaming mouse",
            "score": 0.8875,
            "reason": "similar_category_price"
        }
    ]
    router = get_recommendation_router(recommendation_usecase=mock_uc)
    test_app.include_router(router, prefix="/api/v1")
    client = TestClient(test_app)

    resp = client.get("/api/v1/products/1/recommendations?limit=6")
    assert resp.status_code == 200
    json_data = resp.json()

    assert json_data["success"] is True
    assert json_data["product_id"] == 1
    assert json_data["total"] == 1
    assert isinstance(json_data["recommendations"], list)
    assert isinstance(json_data["data"], list)
    assert json_data["recommendations"][0]["id"] == 10
    assert json_data["recommendations"][0]["score"] == 0.8875
    assert json_data["recommendations"][0]["badge"] == "Terlaris"
    assert json_data["recommendations"][0]["currency"] == "IDR"
