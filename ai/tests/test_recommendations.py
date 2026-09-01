import pytest
from unittest.mock import MagicMock, patch
from fastapi import FastAPI
from fastapi.testclient import TestClient

from app.domain.product import Product
from app.repositories.product_repository import ProductRepository
from app.usecases.recommendation_usecase import RecommendationUseCase
from app.handlers.recommendation_handler import get_recommendation_router, RecommendationResponse


# ==============================================================================
# 1. Unit Tests for RecommendationUseCase
# ==============================================================================

def test_recommendation_usecase_limit_clamping():
    """Verify that RecommendationUseCase enforces limit clamping between 4 and 8 (default 6)"""
    mock_repo = MagicMock(spec=ProductRepository)
    mock_repo.get_similar_products.return_value = [
        {"id": i, "score": 0.9 - (i * 0.05), "reason": "similar_category_price"} for i in range(1, 10)
    ]
    mock_repo.get_frequently_bought_together.return_value = [
        {"id": i, "score": 0.8, "reason": "frequently_bought_together"} for i in range(1, 10)
    ]

    usecase = RecommendationUseCase(product_repo=mock_repo)

    # 1. Lower bound clamping (limit = 2 -> clamped to 4)
    res_low = usecase.get_recommendations(product_id=1, rec_type="similar", limit=2)
    mock_repo.get_similar_products.assert_called_with(product_id=1, limit=4)
    assert len(res_low) == 4

    # 2. Upper bound clamping (limit = 12 -> clamped to 8)
    res_high = usecase.get_recommendations(product_id=1, rec_type="similar", limit=12)
    mock_repo.get_similar_products.assert_called_with(product_id=1, limit=8)
    assert len(res_high) == 8

    # 3. Default value (limit = 6)
    res_default = usecase.get_recommendations(product_id=1, rec_type="similar", limit=6)
    mock_repo.get_similar_products.assert_called_with(product_id=1, limit=6)
    assert len(res_default) == 6

    # 4. None or invalid limit defaults to 6
    res_none = usecase.get_recommendations(product_id=1, rec_type="similar", limit=None)
    mock_repo.get_similar_products.assert_called_with(product_id=1, limit=6)
    assert len(res_none) == 6


def test_recommendation_usecase_type_dispatch():
    """Verify dispatching between 'similar' and 'frequently_bought_together' / 'cross_category'"""
    mock_repo = MagicMock(spec=ProductRepository)
    mock_repo.get_similar_products.return_value = [
        {"id": 2, "score": 0.95, "reason": "similar_category_price"}
    ]
    mock_repo.get_frequently_bought_together.return_value = [
        {"id": 3, "score": 0.85, "reason": "frequently_bought_together"}
    ]

    usecase = RecommendationUseCase(product_repo=mock_repo)

    # Similar items
    res_sim = usecase.get_recommendations(product_id=10, rec_type="similar", limit=6)
    mock_repo.get_similar_products.assert_called_with(product_id=10, limit=6)
    assert res_sim[0]["id"] == 2

    # Frequently bought together aliases
    for rec_type in ["frequently_bought_together", "cross_category", "cart", "fbt"]:
        res_fbt = usecase.get_recommendations(product_id=10, rec_type=rec_type, limit=6)
        mock_repo.get_frequently_bought_together.assert_called_with(product_id=10, limit=6)
        assert res_fbt[0]["id"] == 3


def test_recommendation_usecase_empty_results():
    """Verify handling when repository returns empty list"""
    mock_repo = MagicMock(spec=ProductRepository)
    mock_repo.get_similar_products.return_value = []
    usecase = RecommendationUseCase(product_repo=mock_repo)
    results = usecase.get_recommendations(product_id=999, rec_type="similar", limit=6)
    assert results == []


# ==============================================================================
# 2. Unit Tests for ProductRepository (SQL & Algorithmic Logic)
# ==============================================================================

@patch.object(ProductRepository, "_init_db")
@patch.object(ProductRepository, "_get_connection")
def test_get_similar_products_with_embedding(mock_get_conn, mock_init):
    """Verify pgvector cosine similarity, category boost, and price corridor logic"""
    mock_conn = MagicMock()
    mock_cur = MagicMock()
    mock_get_conn.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cur

    # Target product #10: price = 1,000,000 IDR (corridor = [400,000, 2,500,000])
    mock_cur.fetchone.return_value = {
        "id": 10,
        "name": "Wireless Headphone Pro",
        "category_id": 1,
        "sub_category_id": 2,
        "price": 1000000.0,
        "embedding_str": "[0.12,0.34,0.56]"
    }

    mock_cur.fetchall.return_value = [
        {
            "id": 11,
            "name": "Wireless Headphone Base",
            "sku": "AUD-WNC-011",
            "category_id": 1,
            "sub_category_id": 2,
            "sub_category_name": "Audio",
            "price": 850000.0,
            "currency": "IDR",
            "image_url": "https://example.com/11.jpg",
            "stock_quantity": 15,
            "badge": "Popular",
            "description": "Great audio",
            "score": 0.9250
        },
        {
            "id": 12,
            "name": "Noise Canceling Earbuds",
            "sku": "AUD-EAR-012",
            "category_id": 1,
            "sub_category_id": 3,
            "sub_category_name": "Earphones",
            "price": 1200000.0,
            "currency": "IDR",
            "image_url": "https://example.com/12.jpg",
            "stock_quantity": 20,
            "badge": "",
            "description": "Compact earbuds",
            "score": 0.8650
        }
    ]

    repo = ProductRepository()
    results = repo.get_similar_products(product_id=10, limit=2)

    assert len(results) == 2
    assert results[0]["id"] == 11
    assert results[0]["score"] == 0.9250
    assert results[0]["reason"] == "similar_category_price"
    assert results[1]["id"] == 12

    # Verify SQL query was executed with target embedding and price corridor
    execute_calls = mock_cur.execute.call_args_list
    assert len(execute_calls) >= 2
    target_lookup_call = execute_calls[0]
    assert target_lookup_call[0][1] == (10,)

    vector_query_call = execute_calls[1]
    sql_text = vector_query_call[0][0]
    sql_params = vector_query_call[0][1]

    assert "<=>" in sql_text
    assert "sub_category_id" in sql_text
    assert "category_id" in sql_text
    assert "BETWEEN" in sql_text
    # Check price corridor: 400000.0 <= price <= 2500000.0
    assert sql_params[6] == 400000.0
    assert sql_params[7] == 2500000.0


@patch.object(ProductRepository, "_init_db")
@patch.object(ProductRepository, "_get_connection")
def test_get_similar_products_target_not_found(mock_get_conn, mock_init):
    """Verify that get_similar_products returns empty list when target product does not exist"""
    mock_conn = MagicMock()
    mock_cur = MagicMock()
    mock_get_conn.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cur

    mock_cur.fetchone.return_value = None

    repo = ProductRepository()
    results = repo.get_similar_products(product_id=999, limit=6)

    assert results == []


@patch.object(ProductRepository, "_init_db")
@patch.object(ProductRepository, "_get_connection")
def test_get_similar_products_no_embedding_fallback(mock_get_conn, mock_init):
    """Verify category fallback when target product has no embedding"""
    mock_conn = MagicMock()
    mock_cur = MagicMock()
    mock_get_conn.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cur

    mock_cur.fetchone.return_value = {
        "id": 15,
        "name": "New Product No Emb",
        "category_id": 2,
        "sub_category_id": 4,
        "price": 50000.0,
        "embedding_str": None
    }
    mock_cur.fetchall.return_value = [
        {
            "id": 16,
            "name": "Category Peer Item",
            "sku": "CAT-PEER-016",
            "category_id": 2,
            "sub_category_id": 4,
            "sub_category_name": "Accessories",
            "price": 45000.0,
            "currency": "IDR",
            "image_url": "https://example.com/16.jpg",
            "stock_quantity": 10,
            "badge": "",
            "description": "Peer item",
            "score": 0.5000
        }
    ]

    repo = ProductRepository()
    results = repo.get_similar_products(product_id=15, limit=1)

    assert len(results) == 1
    assert results[0]["id"] == 16
    assert results[0]["reason"] == "category_fallback"


@patch.object(ProductRepository, "_init_db")
@patch.object(ProductRepository, "_get_connection")
def test_get_frequently_bought_together_with_co_occurrence(mock_get_conn, mock_init):
    """Verify co-occurrence aggregation on order_items table"""
    mock_conn = MagicMock()
    mock_cur = MagicMock()
    mock_get_conn.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cur

    mock_cur.fetchall.return_value = [
        {
            "id": 20,
            "co_occurrence_count": 5,
            "name": "Protective Case",
            "sku": "ACC-CSE-020",
            "category_id": 3,
            "sub_category_id": 5,
            "sub_category_name": "Accessories",
            "price": 150000.0,
            "currency": "IDR",
            "image_url": "https://example.com/20.jpg",
            "stock_quantity": 50,
            "badge": "Add-on",
            "description": "Sturdy case"
        },
        {
            "id": 21,
            "co_occurrence_count": 3,
            "name": "Fast Charger 30W",
            "sku": "ACC-CHG-021",
            "category_id": 3,
            "sub_category_id": 6,
            "sub_category_name": "Chargers",
            "price": 250000.0,
            "currency": "IDR",
            "image_url": "https://example.com/21.jpg",
            "stock_quantity": 40,
            "badge": "",
            "description": "USB-C Fast charger"
        }
    ]

    repo = ProductRepository()
    results = repo.get_frequently_bought_together(product_id=1, limit=2)

    assert len(results) == 2
    assert results[0]["id"] == 20
    assert results[0]["reason"] == "frequently_bought_together"
    assert results[0]["score"] == min(1.0, round(0.60 + 0.08 * 5, 4))
    assert results[1]["id"] == 21
    assert results[1]["reason"] == "frequently_bought_together"


@patch.object(ProductRepository, "_init_db")
@patch.object(ProductRepository, "_get_connection")
def test_get_frequently_bought_together_fallback_to_cross_category(mock_get_conn, mock_init):
    """Verify fallback to cross-category vector similarity when co-occurrence data is empty"""
    mock_conn = MagicMock()
    mock_cur = MagicMock()
    mock_get_conn.return_value.__enter__.return_value = mock_conn
    mock_conn.cursor.return_value.__enter__.return_value = mock_cur

    # 1. Co-occurrence query returns empty
    # 2. Target lookup returns target product with embedding
    # 3. Cross-category query returns complementary items
    mock_cur.fetchall.side_effect = [
        [],  # 1. Co-occurrence query: no past orders
        [    # 2. Cross-category query: vector results from other categories
            {
                "id": 30,
                "name": "Laptop Stand Ergonomic",
                "sku": "ACC-STD-030",
                "category_id": 2,  # Different category
                "sub_category_id": 8,
                "sub_category_name": "Office",
                "price": 350000.0,
                "currency": "IDR",
                "image_url": "https://example.com/30.jpg",
                "stock_quantity": 30,
                "badge": "Ergonomic",
                "description": "Aluminum stand",
                "score": 0.8120
            }
        ]
    ]

    mock_cur.fetchone.return_value = {
        "id": 5,
        "category_id": 1,
        "embedding_str": "[0.22,0.44,0.66]"
    }

    repo = ProductRepository()
    results = repo.get_frequently_bought_together(product_id=5, limit=1)

    assert len(results) == 1
    assert results[0]["id"] == 30
    assert results[0]["reason"] == "cross_category_vector"
    assert results[0]["score"] == 0.8120


# ==============================================================================
# 3. Integration Tests with FastAPI TestClient
# ==============================================================================

@pytest.fixture
def test_app():
    """Create a lightweight FastAPI application for handler testing"""
    test_fastapi = FastAPI()
    mock_usecase = MagicMock(spec=RecommendationUseCase)
    router = get_recommendation_router(recommendation_usecase=mock_usecase)
    test_fastapi.include_router(router, prefix="/api/v1")
    return test_fastapi, mock_usecase


def test_api_get_recommendations_similar(test_app):
    """Test GET /api/v1/products/{id}/recommendations with similar items"""
    app_instance, mock_uc = test_app
    client = TestClient(app_instance)

    fake_recs = [
        {
            "id": 2,
            "name": "Wireless Over-Ear Headphones",
            "sku": "AUD-WNC-002",
            "category_id": 1,
            "sub_category_id": 2,
            "sub_category_name": "Audio",
            "price": 1499000.0,
            "currency": "IDR",
            "image_url": "https://example.com/headphones.jpg",
            "stock_quantity": 25,
            "badge": "Terlaris",
            "description": "High fidelity audio",
            "score": 0.892,
            "reason": "similar_category_price"
        },
        {
            "id": 3,
            "name": "Studio Monitor Headphones",
            "sku": "AUD-MON-003",
            "category_id": 1,
            "sub_category_id": 2,
            "sub_category_name": "Audio",
            "price": 1899000.0,
            "currency": "IDR",
            "image_url": "https://example.com/studio.jpg",
            "stock_quantity": 10,
            "badge": "Pro",
            "description": "Flat frequency response",
            "score": 0.845,
            "reason": "similar_category_price"
        }
    ]

    mock_uc.get_recommendations.return_value = fake_recs

    response = client.get("/api/v1/products/1/recommendations?limit=6&type=similar")
    assert response.status_code == 200

    data = response.json()
    assert data["success"] is True
    assert data["product_id"] == 1
    assert data["total"] == 2
    assert len(data["recommendations"]) == 2
    assert data["recommendations"][0]["id"] == 2
    assert data["recommendations"][0]["score"] == 0.892
    assert data["recommendations"][0]["reason"] == "similar_category_price"
    assert data["recommendations"][0]["name"] == "Wireless Over-Ear Headphones"

    # Verify data alias also populated
    assert data["data"] == data["recommendations"]

    mock_uc.get_recommendations.assert_called_once_with(product_id=1, rec_type="similar", limit=6)


def test_api_get_recommendations_cross_category(test_app):
    """Test GET /api/v1/products/{id}/recommendations with cross_category/fbt strategy"""
    app_instance, mock_uc = test_app
    client = TestClient(app_instance)

    fake_recs = [
        {
            "id": 20,
            "name": "Headphone Stand Aluminum",
            "sku": "ACC-STD-020",
            "category_id": 3,
            "sub_category_id": 7,
            "sub_category_name": "Accessories",
            "price": 250000.0,
            "currency": "IDR",
            "image_url": "https://example.com/stand.jpg",
            "stock_quantity": 30,
            "badge": "Essential",
            "description": "Desk organizer stand",
            "score": 0.78,
            "reason": "cross_category_vector"
        }
    ]

    mock_uc.get_recommendations.return_value = fake_recs

    response = client.get("/api/v1/products/1/recommendations?limit=4&type=cross_category")
    assert response.status_code == 200

    data = response.json()
    assert data["success"] is True
    assert data["product_id"] == 1
    assert data["total"] == 1
    assert data["recommendations"][0]["id"] == 20
    assert data["recommendations"][0]["reason"] == "cross_category_vector"

    mock_uc.get_recommendations.assert_called_once_with(product_id=1, rec_type="cross_category", limit=4)


def test_api_get_recommendations_rec_type_alias(test_app):
    """Test GET /api/v1/products/{id}/recommendations with rec_type query parameter alias"""
    app_instance, mock_uc = test_app
    client = TestClient(app_instance)

    fake_recs = [
        {
            "id": 5,
            "name": "USB-C Audio DAC",
            "sku": "ACC-DAC-005",
            "category_id": 3,
            "sub_category_id": 9,
            "sub_category_name": "Adapters",
            "price": 199000.0,
            "currency": "IDR",
            "image_url": "https://example.com/dac.jpg",
            "stock_quantity": 18,
            "badge": "Hi-Res",
            "description": "High resolution DAC",
            "score": 0.81,
            "reason": "frequently_bought_together"
        }
    ]

    mock_uc.get_recommendations.return_value = fake_recs

    response = client.get("/api/v1/products/2/recommendations?rec_type=frequently_bought_together")
    assert response.status_code == 200

    data = response.json()
    assert data["success"] is True
    assert data["product_id"] == 2
    assert len(data["recommendations"]) == 1

    mock_uc.get_recommendations.assert_called_once_with(product_id=2, rec_type="frequently_bought_together", limit=6)


def test_api_get_recommendations_invalid_id(test_app):
    """Test GET /api/v1/products/{id}/recommendations with non-positive product ID"""
    app_instance, _ = test_app
    client = TestClient(app_instance)
    response = client.get("/api/v1/products/0/recommendations")
    assert response.status_code == 422  # Validation error for ge=1


def test_api_get_recommendations_server_error(test_app):
    """Test GET /api/v1/products/{id}/recommendations when usecase raises an exception"""
    app_instance, mock_uc = test_app
    client = TestClient(app_instance)
    mock_uc.get_recommendations.side_effect = RuntimeError("Database connection lost")

    response = client.get("/api/v1/products/1/recommendations")
    assert response.status_code == 500
    assert "Database connection lost" in response.json()["detail"]


def test_main_app_has_recommendation_route():
    """Verify that main FastAPI app includes the recommendations endpoint"""
    with patch("psycopg2.connect"), \
         patch("redis.Redis"):
        import sys
        if "app.main" in sys.modules:
            del sys.modules["app.main"]
        from app.main import app as main_fastapi_app

        openapi_schema = main_fastapi_app.openapi()
        paths = list(openapi_schema.get("paths", {}).keys())
        assert "/api/v1/products/{id}/recommendations" in paths

