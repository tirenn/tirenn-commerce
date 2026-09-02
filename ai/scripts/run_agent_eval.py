"""
Standalone Live CLI Benchmark Runner for Tirenn AI Agent Evaluation
Runs golden benchmark test cases against the live AI Shopper Agent and Ollama LLM,
printing a colored terminal scorecard table.
"""

import sys
import time
import json
from pathlib import Path

# Add project root to sys.path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from app.eval.runner import AgentEvalRunner
from app.harness.agent import ShopperAgent
from app.repositories.llm_repository import LLMRepository
from app.repositories.product_repository import ProductRepository
from app.repositories.knowledge_repository import KnowledgeRepository
from app.repositories.embedding_repository import EmbeddingRepository
from app.usecases.knowledge_usecase import KnowledgeUseCase


def main():
    print("\n" + "=" * 80)
    print("🚀 TIRENN AI AGENT EVALUATION (AGENT EVAL) BENCHMARK SUITE")
    print("=" * 80 + "\n")

    print("📦 Initializing repositories and Shopper Agent...")
    try:
        embedding_repo = EmbeddingRepository()
        product_repo = ProductRepository()
        knowledge_repo = KnowledgeRepository()
        knowledge_uc = KnowledgeUseCase(knowledge_repo, embedding_repo)
        llm_repo = LLMRepository()

        agent = ShopperAgent(
            llm_repo=llm_repo,
            product_repo=product_repo,
            knowledge_usecase=knowledge_uc,
            backend_api_url=None
        )
    except Exception as e:
        print(f"❌ Failed to initialize agent components: {e}")
        return

    runner = AgentEvalRunner(embedding_encoder=embedding_repo)
    dataset = runner.load_dataset()

    print(f"📋 Loaded {len(dataset)} Golden Evaluation Cases from dataset.\n")

    results = []
    for idx, case in enumerate(dataset, 1):
        test_id = case.get("id")
        category = case.get("category")
        prompt = case.get("prompt")

        print(f"[{idx}/{len(dataset)}] Evaluating `{test_id}` ({category})...")
        print(f"     Prompt: \"{prompt}\"")

        start = time.perf_counter()
        try:
            # Execute Agent ReAct turn
            agent_res = agent.chat(
                messages=[{"role": "user", "content": prompt}],
                session_id=f"eval-{test_id}"
            )
            elapsed_ms = (time.perf_counter() - start) * 1000.0

            output_payload = {
                "reply": agent_res.reply,
                "tool_calls": agent_res.tool_calls,
                "iterations_count": agent_res.iterations_count,
            }

            eval_res = runner.evaluate_turn(case, output_payload, latency_ms=elapsed_ms)
            results.append(eval_res)

            status_icon = "✅ PASS" if eval_res.passed else "❌ FAIL"
            print(f"     Status: {status_icon} | Score: {eval_res.total_score:.2f} | Latency: {elapsed_ms:.0f}ms")
            if not eval_res.passed:
                for score in eval_res.scores:
                    if not score.passed:
                        print(f"       ⚠️ {score.metric_name}: {score.reason}")
        except Exception as e:
            elapsed_ms = (time.perf_counter() - start) * 1000.0
            print(f"     ❌ Execution Error: {e}")

        print("-" * 80)

    # Print Final Markdown Report
    print("\n" + runner.generate_report(results) + "\n")


if __name__ == "__main__":
    main()
