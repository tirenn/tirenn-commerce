"""
Pytest Agent Evaluation Suite for Tirenn Commerce AI Service
Runs deterministic trajectory assertions, parameter precision checks, and safety evaluations.
"""

import pytest
from app.eval.metrics import (
    ToolMatchMetric,
    ParameterPrecisionMetric,
    TrajectoryEfficiencyMetric,
    NegativeConstraintMetric,
    EmbeddingSimilarityMetric,
)
from app.eval.runner import AgentEvalRunner


def test_tool_match_metric():
    """Verify tool match evaluator correctly matches single and multiple tools"""
    calls = [{"name": "search_products", "args": {"category_id": 1}}]
    
    # Matching tool
    score1 = ToolMatchMetric.evaluate(calls, expected_tools=["search_products"])
    assert score1.passed is True
    assert score1.score == 1.0

    # Non-matching tool
    score2 = ToolMatchMetric.evaluate(calls, expected_tools=["adjust_product_stock"])
    assert score2.passed is False
    assert score2.score == 0.0


def test_parameter_precision_metric():
    """Verify argument range and exact matching"""
    calls = [
        {
            "name": "search_products",
            "args": {
                "category_id": 1,
                "max_price": 4500000,
                "in_stock": True
            }
        }
    ]

    # Valid arguments within budget
    score1 = ParameterPrecisionMetric.evaluate(calls, expected_args={"category_id": 1, "max_price": 5000000})
    assert score1.passed is True
    assert score1.score == 1.0

    # Out of range price
    score2 = ParameterPrecisionMetric.evaluate(calls, expected_args={"category_id": 1, "max_price": 3000000})
    assert score2.passed is False


def test_negative_constraints_metric():
    """Verify negative constraints properly detect forbidden brand violations"""
    clean_reply = "Berikut rekomendasi Samsung Galaxy S24 dan iPhone 16 Pro."
    bad_reply = "Berikut rekomendasi Xiaomi 14 Ultra dan Poco F6."

    # Passed when forbidden keywords absent
    score1 = NegativeConstraintMetric.evaluate(clean_reply, forbidden_keywords=["Xiaomi", "Poco"])
    assert score1.passed is True
    assert score1.score == 1.0

    # Failed when forbidden keyword detected
    score2 = NegativeConstraintMetric.evaluate(bad_reply, forbidden_keywords=["Xiaomi", "Poco"])
    assert score2.passed is False
    assert "Violated negative constraint" in score2.reason


def test_trajectory_efficiency_metric():
    """Verify loop step limits prevent runaway iterations"""
    score1 = TrajectoryEfficiencyMetric.evaluate(iteration_count=1, max_allowed_iterations=3)
    assert score1.passed is True
    assert score1.score == 1.0

    score2 = TrajectoryEfficiencyMetric.evaluate(iteration_count=5, max_allowed_iterations=3)
    assert score2.passed is False


def test_embedding_similarity_metric():
    """Verify mathematical vector cosine similarity evaluation"""
    v1 = [1.0, 0.0, 0.0]
    v2 = [1.0, 0.0, 0.0]
    v3 = [0.0, 1.0, 0.0]

    score_identical = EmbeddingSimilarityMetric.evaluate(v1, v2, threshold=0.70)
    assert score_identical.passed is True
    assert score_identical.score == 1.0

    score_orthogonal = EmbeddingSimilarityMetric.evaluate(v1, v3, threshold=0.70)
    assert score_orthogonal.passed is False
    assert score_orthogonal.score == 0.0


def test_agent_eval_runner_dataset_execution():
    """Verify end-to-end evaluation runner loads dataset and produces structured reports"""
    runner = AgentEvalRunner()
    dataset = runner.load_dataset()
    assert len(dataset) >= 5, "Evaluation dataset should contain at least 5 benchmark cases"

    # Simulate evaluation on first case
    case = dataset[0]
    simulated_output = {
        "reply": "Berikut 2 rekomendasi smartphone gaming terbaik: ROG Phone 8 dan Samsung Galaxy S24 Ultra.",
        "tool_calls": [{"name": "search_products", "args": {"category_id": 1}}],
        "iterations_count": 1
    }

    result = runner.evaluate_turn(case, simulated_output, latency_ms=1250.0)
    assert result.passed is True
    assert result.total_score >= 0.90

    # Report formatting
    report = runner.generate_report([result])
    assert "Tirenn AI Agent Evaluation Scorecard" in report
    assert "SHOP-CAT-01" in report
    assert "PASS" in report
