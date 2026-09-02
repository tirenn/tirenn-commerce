import logging
from typing import List, Optional, Dict, Any

from app.core.config import settings
from app.core.prompt_loader import load_prompt
from app.domain.chat import ChatMessage, ChatShopperResult
from app.repositories.llm_repository import LLMRepository
from app.repositories.product_repository import ProductRepository
from app.repositories.analytics_repository import AnalyticsRepository
from app.usecases.knowledge_usecase import KnowledgeUseCase
from app.repositories.session_repository import SessionRepository
from app.harness.agent import AgentHarness
from app.harness.tools.admin import (
    GetExecutiveDashboardMetricsTool,
    GetRecentOrdersOverviewTool,
    GetLowStockProductsTool,
    AdjustProductStockTool,
    SearchAdminInternalSOPTool,
)

logger = logging.getLogger("ai-service.usecase.admin")


class AdminUseCase:
    """Enterprise Admin AI Copilot UseCase with strict tool isolation and direct repository access"""

    def __init__(
        self,
        llm_repo: LLMRepository,
        product_repo: ProductRepository,
        knowledge_usecase: KnowledgeUseCase,
        analytics_repo: Optional[AnalyticsRepository] = None,
        session_repo: Optional[SessionRepository] = None,
        system_prompt: Optional[str] = None
    ):
        self.llm_repo = llm_repo
        self.product_repo = product_repo
        self.analytics_repo = analytics_repo or AnalyticsRepository()
        self.knowledge_usecase = knowledge_usecase
        self.session_repo = session_repo or SessionRepository()
        self.system_prompt = system_prompt or load_prompt("admin_agent.md")

        # Initialize Admin-exclusive tools with direct repository injection (Zero HTTP API calls)
        self.tools = [
            GetExecutiveDashboardMetricsTool(analytics_repo=self.analytics_repo),
            GetRecentOrdersOverviewTool(analytics_repo=self.analytics_repo),
            GetLowStockProductsTool(product_repo=self.product_repo),
            AdjustProductStockTool(product_repo=self.product_repo),
            SearchAdminInternalSOPTool(knowledge_usecase=self.knowledge_usecase),
        ]

        self.agent = AgentHarness(
            llm_repo=self.llm_repo,
            tools=self.tools,
            system_prompt=self.system_prompt,
            max_iterations=settings.MAX_AGENT_ITERATIONS
        )

    async def chat(
        self,
        messages: List[ChatMessage],
        admin_claims: Dict[str, Any],
        session_id: Optional[str] = None,
        token: Optional[str] = None
    ) -> Dict[str, Any]:
        """Execute Admin AI Copilot conversation turn with Redis sliding window history and native LLM reasoning"""
        if not messages:
            return {"reply": "Halo Admin! Ada yang bisa saya bantu terkait operasional toko, metrik omzet, stok, atau SOP gudang?", "tool_calls": []}

        latest_user_message = messages[-1]
        active_session_id = session_id or f"admin_session_default_{admin_claims.get('sub', 1)}"
        user_content = latest_user_message.content if hasattr(latest_user_message, 'content') else latest_user_message.get("content", "")

        # 1. Fetch past conversation history from Redis List
        past_history = self.session_repo.get_history(session_id=active_session_id, limit=settings.SESSION_HISTORY_LIMIT)

        # 2. Build full message sequence
        conversation_sequence: List[Dict[str, str]] = []
        for h in past_history:
            role = h.role if hasattr(h, 'role') else h.get("role", "user")
            content = h.content if hasattr(h, 'content') else h.get("content", "")
            conversation_sequence.append({"role": role, "content": content})

        # Append incoming user message if not already trailing
        user_role = latest_user_message.role if hasattr(latest_user_message, 'role') else latest_user_message.get("role", "user")
        if not conversation_sequence or conversation_sequence[-1]["content"] != user_content:
            conversation_sequence.append({"role": user_role, "content": user_content})

        # 3. Prepare execution context with Admin JWT token
        context = {
            "is_admin": True,
            "admin_email": admin_claims.get("email", "admin@gocommerce.com"),
            "admin_id": admin_claims.get("sub", 1),
            "token": token or ""
        }

        # 4. Run Admin Agent Harness (Model determines tool calls naturally from conversation history)
        agent_result = await self.agent.run(
            messages=conversation_sequence,
            context=context
        )

        reply_text = getattr(agent_result, "reply", "") or ""
        tool_calls = getattr(agent_result, "tool_calls", []) or []

        # 5. Atomically persist conversation pair to Redis
        self.session_repo.append_messages(
            session_id=active_session_id,
            messages=[
                {"role": "user", "content": user_content},
                {"role": "assistant", "content": reply_text}
            ]
        )

        return {
            "reply": reply_text,
            "tool_calls": tool_calls,
            "session_id": active_session_id,
            "admin_email": admin_claims.get("email")
        }
