import logging
from typing import Dict, Any, Optional
from app.harness.tools.base import BaseTool

logger = logging.getLogger("ai-service.harness.tools.admin.knowledge")


class SearchAdminInternalSOPTool(BaseTool):
    """Admin Tool for querying internal merchant operations, warehouse logistics, stock audit, and escalation SOPs"""

    name = "search_admin_internal_sop"
    description = "Search internal admin/merchant operational procedures, warehouse picking/packing guidelines, inventory audit protocols, and courier escalation SOPs from confidential admin knowledge documents."
    parameters_schema = {
        "type": "object",
        "properties": {
            "query": {
                "type": "string",
                "description": "Question or topic regarding internal admin operations, warehouse procedures, stock adjustments audit, or merchant guidelines (e.g. 'SOP picking dan packing gudang', 'prosedur audit stok selisih', 'klaim logistik'). Required."
            }
        },
        "required": ["query"]
    }

    def __init__(self, knowledge_usecase):
        self.knowledge_usecase = knowledge_usecase

    async def execute(self, args: Dict[str, Any], context: Optional[Dict[str, Any]] = None) -> Dict[str, Any]:
        query = (args.get("query") or "").strip()
        logger.info(f"🔒 [ADMIN TOOL: search_admin_internal_sop] query='{query}' (doc_type='SOP_ADMIN')")

        if not self.knowledge_usecase:
            return {"status": "error", "message": "Knowledge Base is not initialized."}

        results = self.knowledge_usecase.query_knowledge(
            query=query,
            limit=4,
            score_threshold=0.15,
            doc_type="SOP_ADMIN"
        )
        if not results:
            return {"status": "not_found", "message": f"No internal admin SOP found for query '{query}'."}

        formatted = [
            {
                "document": r.get("document_title"),
                "page": r.get("page_number"),
                "content": f"<confidential_admin_document>\n{r.get('content', '').strip()}\n</confidential_admin_document>",
                "relevance_score": round(r.get("score", 0.0), 3)
            }
            for r in results
        ]

        return {
            "status": "found",
            "found_count": len(formatted),
            "internal_sop_excerpts": formatted
        }
