"""
Agent Evaluation (Agent Eval) Framework for Tirenn Commerce AI
Provides 3-Tier evaluation:
- Tier 1: Deterministic Trajectory & Parameter Assertions
- Tier 2: Vector Embedding Semantic Alignment
- Tier 3: LLM-as-a-Judge RAG Groundedness & Persona Evaluation
"""

from app.eval.metrics import (
    EvaluationResult,
    ToolMatchMetric,
    ParameterPrecisionMetric,
    TrajectoryEfficiencyMetric,
    NegativeConstraintMetric,
    EmbeddingSimilarityMetric,
)
from app.eval.llm_judge import LLMJudgeEvaluator, JudgeResult
from app.eval.runner import AgentEvalRunner

__all__ = [
    "EvaluationResult",
    "ToolMatchMetric",
    "ParameterPrecisionMetric",
    "TrajectoryEfficiencyMetric",
    "NegativeConstraintMetric",
    "EmbeddingSimilarityMetric",
    "LLMJudgeEvaluator",
    "JudgeResult",
    "AgentEvalRunner",
]
