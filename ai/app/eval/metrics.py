import re
import math
from dataclasses import dataclass, field
from typing import Dict, Any, List, Optional
import numpy as np


@dataclass
class MetricScore:
    metric_name: str
    passed: bool
    score: float  # 0.0 to 1.0
    reason: str
    details: Dict[str, Any] = field(default_factory=dict)


@dataclass
class EvaluationResult:
    test_id: str
    category: str
    prompt: str
    passed: bool
    total_score: float
    latency_ms: float
    scores: List[MetricScore]
    model_reply: str
    tool_calls: List[Dict[str, Any]]
    iterations_count: int


class ToolMatchMetric:
    """Evaluates whether the agent invoked the expected tool(s)"""

    @staticmethod
    def evaluate(actual_tool_calls: List[Dict[str, Any]], expected_tools: List[str]) -> MetricScore:
        if not expected_tools:
            return MetricScore(
                metric_name="ToolMatch",
                passed=True,
                score=1.0,
                reason="No specific tools expected (direct response)."
            )

        actual_tool_names = [call.get("name") for call in actual_tool_calls if isinstance(call, dict)]
        
        # Check if at least one expected tool was invoked
        matched = [tool for tool in expected_tools if tool in actual_tool_names]
        match_rate = len(matched) / len(expected_tools) if expected_tools else 1.0
        passed = len(matched) > 0

        return MetricScore(
            metric_name="ToolMatch",
            passed=passed,
            score=match_rate,
            reason=f"Matched {len(matched)}/{len(expected_tools)} expected tools: {matched} vs actual {actual_tool_names}",
            details={"actual": actual_tool_names, "expected": expected_tools, "matched": matched}
        )


class ParameterPrecisionMetric:
    """Evaluates whether tool call arguments match expected types, bounds, and values"""

    @staticmethod
    def evaluate(
        actual_tool_calls: List[Dict[str, Any]],
        expected_args: Dict[str, Any]
    ) -> MetricScore:
        if not expected_args:
            return MetricScore(
                metric_name="ParameterPrecision",
                passed=True,
                score=1.0,
                reason="No argument constraints defined."
            )

        if not actual_tool_calls:
            return MetricScore(
                metric_name="ParameterPrecision",
                passed=False,
                score=0.0,
                reason="No tool calls executed to validate arguments."
            )

        # Aggregate all arguments across executed tools
        all_actual_args: Dict[str, Any] = {}
        for call in actual_tool_calls:
            args = call.get("args") or call.get("arguments") or {}
            if isinstance(args, dict):
                all_actual_args.update(args)

        passed_checks = 0
        total_checks = len(expected_args)
        failed_reasons = []

        for key, expected_val in expected_args.items():
            if key not in all_actual_args:
                failed_reasons.append(f"Missing expected argument '{key}'")
                continue

            actual_val = all_actual_args[key]

            # Range check for prices / numbers
            if "max_" in key or "min_" in key or "limit" in key:
                try:
                    act_num = float(actual_val)
                    exp_num = float(expected_val)
                    if "max_" in key and act_num <= exp_num:
                        passed_checks += 1
                    elif "min_" in key and act_num >= exp_num:
                        passed_checks += 1
                    elif act_num == exp_num:
                        passed_checks += 1
                    else:
                        failed_reasons.append(f"Argument '{key}': expected {expected_val}, got {actual_val}")
                except (ValueError, TypeError):
                    failed_reasons.append(f"Argument '{key}': could not cast '{actual_val}' to float")
            else:
                # Exact match or case-insensitive string match
                if str(actual_val).lower() == str(expected_val).lower():
                    passed_checks += 1
                else:
                    failed_reasons.append(f"Argument '{key}': expected '{expected_val}', got '{actual_val}'")

        score = passed_checks / total_checks if total_checks > 0 else 1.0
        passed = (score >= 0.8)

        return MetricScore(
            metric_name="ParameterPrecision",
            passed=passed,
            score=score,
            reason="All arguments valid" if passed else "; ".join(failed_reasons),
            details={"actual_args": all_actual_args, "expected_args": expected_args}
        )


class TrajectoryEfficiencyMetric:
    """Evaluates whether the ReAct loop completed within acceptable iteration limits"""

    @staticmethod
    def evaluate(iteration_count: int, max_allowed_iterations: int = 3) -> MetricScore:
        passed = (iteration_count <= max_allowed_iterations)
        # 1 step = 1.0, 2 steps = 0.9, 3 steps = 0.8, >3 steps penalty
        if iteration_count <= 1:
            score = 1.0
        elif iteration_count <= 2:
            score = 0.9
        elif iteration_count <= max_allowed_iterations:
            score = 0.8
        else:
            score = max(0.0, 1.0 - (iteration_count - max_allowed_iterations) * 0.3)

        return MetricScore(
            metric_name="TrajectoryEfficiency",
            passed=passed,
            score=score,
            reason=f"Agent finished in {iteration_count} iterations (limit: {max_allowed_iterations})",
            details={"iterations": iteration_count, "limit": max_allowed_iterations}
        )


class NegativeConstraintMetric:
    """Evaluates whether the agent strictly respected forbidden keywords and brand exclusions"""

    @staticmethod
    def evaluate(
        model_reply: str,
        forbidden_keywords: List[str],
        must_include_keywords: Optional[List[str]] = None
    ) -> MetricScore:
        reply_lower = model_reply.lower() if model_reply else ""
        violations = []

        if forbidden_keywords:
            for forbidden in forbidden_keywords:
                pattern = r'\b' + re.escape(forbidden.lower()) + r'\b'
                if re.search(pattern, reply_lower):
                    violations.append(f"Violated negative constraint: found '{forbidden}'")

        missing_required = []
        if must_include_keywords:
            for req in must_include_keywords:
                pattern = r'\b' + re.escape(req.lower()) + r'\b'
                if not re.search(pattern, reply_lower):
                    missing_required.append(f"Missing required keyword: '{req}'")

        passed = (len(violations) == 0 and len(missing_required) == 0)
        score = 1.0 if passed else (0.5 if len(violations) == 0 else 0.0)

        all_issues = violations + missing_required
        return MetricScore(
            metric_name="NegativeConstraints",
            passed=passed,
            score=score,
            reason="Negative and positive constraints satisfied" if passed else "; ".join(all_issues),
            details={"violations": violations, "missing_required": missing_required}
        )


class EmbeddingSimilarityMetric:
    """Evaluates mathematical cosine similarity between model answer and golden ground truth intent using embeddings"""

    @staticmethod
    def cosine_similarity(v1: List[float], v2: List[float]) -> float:
        a = np.array(v1, dtype=np.float32)
        b = np.array(v2, dtype=np.float32)
        norm_a = np.linalg.norm(a)
        norm_b = np.linalg.norm(b)
        if norm_a == 0 or norm_b == 0:
            return 0.0
        return float(np.dot(a, b) / (norm_a * norm_b))

    @classmethod
    def evaluate(
        cls,
        actual_vec: Optional[List[float]],
        golden_vec: Optional[List[float]],
        threshold: float = 0.70
    ) -> MetricScore:
        if actual_vec is None or golden_vec is None:
            return MetricScore(
                metric_name="EmbeddingSimilarity",
                passed=True,
                score=1.0,
                reason="Embedding comparison skipped (vectors not provided)."
            )

        sim = cls.cosine_similarity(actual_vec, golden_vec)
        passed = (sim >= threshold)

        return MetricScore(
            metric_name="EmbeddingSimilarity",
            passed=passed,
            score=max(0.0, min(1.0, sim)),
            reason=f"Cosine similarity: {sim:.3f} (threshold: {threshold:.2f})",
            details={"similarity": sim, "threshold": threshold}
        )
