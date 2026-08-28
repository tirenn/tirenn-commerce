import logging
from typing import List, Dict, Any, Optional

logger = logging.getLogger("ai-service.harness.guardrails.safety")

class SafetyGuardrail:
    """Safety and Execution Guardrail for Agent Harness"""

    def __init__(self, max_iterations: int = 5, max_budget_tools: int = 8):
        self.max_iterations = max_iterations
        self.max_budget_tools = max_budget_tools

    def check_loop_budget(self, iteration: int, tool_call_count: int) -> bool:
        """Verify whether agent is within allowed execution budget"""
        if iteration > self.max_iterations:
            logger.warning(f"⚠️ [SAFETY_GUARDRAIL] Execution exceeded max iterations ({self.max_iterations})")
            return False
        if tool_call_count > self.max_budget_tools:
            logger.warning(f"⚠️ [SAFETY_GUARDRAIL] Execution exceeded max tool calls ({self.max_budget_tools})")
            return False
        return True

    def sanitize_output_text(self, text: str) -> str:
        """Remove markdown image injections or hallucinated URLs from assistant responses"""
        import re
        # Strip markdown images ![](...)
        clean_text = re.sub(r'!\[.*?\]\(.*?\)', '', text)
        return clean_text.strip()
