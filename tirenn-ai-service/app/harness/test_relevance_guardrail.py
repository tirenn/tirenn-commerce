from app.harness.guardrails.relevance import RelevanceGuardrail

def test_celana_panjang_pruning():
    guardrail = RelevanceGuardrail()

    # Simulated candidate products from vector search for query 'celana panjang pria'
    raw_products = [
        {
            "id": 9,
            "name": "Celana Jeans Pria Slim Fit Denim Melar Indigo Gelap",
            "sku": "ID-MPN-001",
            "sub_category_name": "Celana & Jeans Pria"
        },
        {
            "id": 10,
            "name": "Tailored Slim-Fit Stretch Denim Jeans in Deep Indigo",
            "sku": "EN-MPN-001",
            "sub_category_name": "Celana & Jeans Pria"
        },
        {
            "id": 45,
            "name": "Celana Pendek Tactical Quickdry Outdoor Pria Abu",
            "sku": "ID-MPN-005",
            "sub_category_name": "Celana & Jeans Pria"
        },
        {
            "id": 46,
            "name": "Quick-Dry Ripstop Cargo Shorts with Utility Pockets",
            "sku": "EN-MPN-005",
            "sub_category_name": "Celana & Jeans Pria"
        }
    ]

    filtered = guardrail.filter_products("celana panjang", raw_products)
    filtered_names = [p["name"] for p in filtered]

    # Should keep long pants / jeans
    assert "Celana Jeans Pria Slim Fit Denim Melar Indigo Gelap" in filtered_names
    assert "Tailored Slim-Fit Stretch Denim Jeans in Deep Indigo" in filtered_names

    # Should prune celana pendek and shorts
    assert "Celana Pendek Tactical Quickdry Outdoor Pria Abu" not in filtered_names
    assert "Quick-Dry Ripstop Cargo Shorts with Utility Pockets" not in filtered_names
    assert len(filtered) == 2
    print("✅ test_celana_panjang_pruning PASSED!")

def test_gender_pruning():
    guardrail = RelevanceGuardrail()

    raw_products = [
        {"id": 1, "name": "Sepatu Sneakers Kanvas Pria Hitam", "sku": "ID-MSH-001"},
        {"id": 2, "name": "Flat Shoes Balet Lipat Wanita Nude", "sku": "ID-WSH-001"}
    ]

    filtered = guardrail.filter_products("sepatu pria", raw_products)
    filtered_names = [p["name"] for p in filtered]

    assert "Sepatu Sneakers Kanvas Pria Hitam" in filtered_names
    assert "Flat Shoes Balet Lipat Wanita Nude" not in filtered_names
    print("✅ test_gender_pruning PASSED!")

if __name__ == "__main__":
    test_celana_panjang_pruning()
    test_gender_pruning()
    print("🎉 ALL RELEVANCE GUARDRAIL TESTS PASSED!")
