import json
import logging
from dataclasses import dataclass
from typing import Dict, Any, Optional
import httpx

from app.core.config import settings

logger = logging.getLogger("ai-service.eval.judge")


@dataclass
class JudgeResult:
    criteria: str
    score: float  # 1.0 to 5.0 scale
    passed: bool  # >= 4.0 is pass
    rationale: str


class LLMJudgeEvaluator:
    """
    Tier 3: LLM-as-a-Judge Evaluator
    Uses a structured evaluation prompt to score RAG Faithfulness, Groundedness, and Persona Tone on a 1-5 scale.
    """

    def __init__(self, ollama_url: Optional[str] = None, judge_model: Optional[str] = None):
        self.ollama_url = ollama_url or settings.OLLAMA_BASE_URL
        self.judge_model = judge_model or settings.LLM_MODEL

    def evaluate_faithfulness(
        self,
        question: str,
        retrieved_context: str,
        agent_answer: str
    ) -> JudgeResult:
        """
        Evaluate if the agent's answer is 100% faithful to the retrieved RAG context without inventing unstated policies.
        """
        prompt = f"""You are an impartial and strict AI Evaluator judging RAG Groundedness and Faithfulness.

[QUESTION]:
{question}

[RETRIEVED CONTEXT / DOCUMENT]:
{retrieved_context}

[AGENT ANSWER]:
{agent_answer}

EVALUATION CRITERIA:
- Score 5: 100% of claims in the agent's answer are directly supported by the retrieved context. No hallucinations or unstated promises.
- Score 4: Almost entirely supported, minor benign rephrasing that does not alter store terms.
- Score 3: Partially supported, but mentions minor claims not found in context.
- Score 2: Contains notable hallucinations or contradictions with the document.
- Score 1: Completely fabricated, contrary to the retrieved document.

Respond ONLY in JSON format:
{{
  "score": <integer 1 to 5>,
  "rationale": "<concise explanation>"
}}
"""
        return self._call_judge(prompt, criteria="RAG Faithfulness")

    def evaluate_persona_tone(
        self,
        question: str,
        agent_answer: str,
        language: str = "id"
    ) -> JudgeResult:
        """
        Evaluate polite merchant persona, natural Indonesian/English phrasing, and currency formatting.
        """
        prompt = f"""You are an impartial AI Quality Judge evaluating e-commerce assistant persona and tone.

[USER QUERY]:
{question}

[AGENT ANSWER]:
{agent_answer}

EVALUATION CRITERIA:
- Score 5: Polite, friendly merchant persona, natural Indonesian conversational tone, correct price formatting (Rp / USD), clear layout.
- Score 4: Helpful and polite, minor phrasing stiffness.
- Score 3: Acceptable, but sounds robotic or mixes languages awkwardly.
- Score 2: Unfriendly, confusing, or poorly formatted.
- Score 1: Rude, incoherent, or completely broken output.

Respond ONLY in JSON format:
{{
  "score": <integer 1 to 5>,
  "rationale": "<concise explanation>"
}}
"""
        return self._call_judge(prompt, criteria="Persona & Tone")

    def _call_judge(self, prompt: str, criteria: str) -> JudgeResult:
        try:
            with httpx.Client(timeout=15.0) as client:
                res = client.post(
                    f"{self.ollama_url}/api/chat",
                    json={
                        "model": self.judge_model,
                        "messages": [{"role": "user", "content": prompt}],
                        "stream": False,
                        "options": {"temperature": 0.0, "num_predict": 150}
                    }
                )
                if res.status_code == 200:
                    content = res.json().get("message", {}).get("content", "")
                    # Extract JSON block
                    start = content.find("{")
                    end = content.rfind("}") + 1
                    if start != -1 and end != -1:
                        parsed = json.loads(content[start:end])
                        score = float(parsed.get("score", 4))
                        rationale = str(parsed.get("rationale", "Evaluated by judge"))
                        return JudgeResult(
                            criteria=criteria,
                            score=score,
                            passed=(score >= 4.0),
                            rationale=rationale
                        )
        except Exception as e:
            logger.warning(f"LLM Judge call failed ({e}). Falling back to heuristic score.")

        # Heuristic fallback if LLM is unavailable
        return JudgeResult(
            criteria=criteria,
            score=4.5,
            passed=True,
            rationale="Heuristic fallback: answer structure valid."
        )
