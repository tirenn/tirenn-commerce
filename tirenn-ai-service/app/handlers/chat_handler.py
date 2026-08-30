import logging
from typing import List, Dict, Any, Optional
from pydantic import BaseModel
from fastapi import APIRouter, HTTPException, Depends, Request

from app.domain.chat import ChatMessage
from app.usecases.shopper_usecase import ShopperUseCase
from app.usecases.admin_usecase import AdminUseCase
from app.core.security import verify_admin_jwt

logger = logging.getLogger("ai-service.handler.chat")


class ChatShopperRequest(BaseModel):
    messages: List[ChatMessage]
    is_authenticated: Optional[bool] = False
    user_name: Optional[str] = None
    session_id: Optional[str] = None
    cart_items: Optional[List[Dict[str, Any]]] = None


class ChatShopperResponse(BaseModel):
    success: bool
    reply: str
    tool_calls: List[Dict[str, Any]] = []
    suggested_products: List[Dict[str, Any]] = []
    cart_action: Optional[Dict[str, Any]] = None


class ChatAdminRequest(BaseModel):
    messages: List[ChatMessage]
    session_id: Optional[str] = None


class ChatAdminResponse(BaseModel):
    success: bool
    reply: str
    tool_calls: List[Dict[str, Any]] = []
    session_id: Optional[str] = None
    admin_email: Optional[str] = None


def get_chat_router(shopper_usecase: ShopperUseCase, admin_usecase: Optional[AdminUseCase] = None) -> APIRouter:
    """Factory creating chat routes with injected ShopperUseCase and AdminUseCase"""
    router = APIRouter(tags=["Chat AI Assistant"])

    async def _handle_shopper_chat(req: ChatShopperRequest) -> ChatShopperResponse:
        try:
            result = await shopper_usecase.chat(
                messages=req.messages,
                is_authenticated=req.is_authenticated or False,
                user_name=req.user_name,
                session_id=req.session_id,
                cart_items=req.cart_items
            )
            return ChatShopperResponse(
                success=True,
                reply=result.reply,
                tool_calls=result.tool_calls,
                suggested_products=result.suggested_products[:6],
                cart_action=result.cart_action
            )
        except Exception as e:
            logger.error(f"ChatShopper handler exception: {e}", exc_info=True)
            raise HTTPException(status_code=500, detail=str(e))

    @router.post("/chat/shopper", response_model=ChatShopperResponse)
    async def chat_shopper(req: ChatShopperRequest):
        return await _handle_shopper_chat(req)

    @router.post("/chat", response_model=ChatShopperResponse)
    async def chat(req: ChatShopperRequest):
        return await _handle_shopper_chat(req)

    @router.delete("/chat/session/{session_id}")
    async def delete_session(session_id: str):
        """Delete customer chat session history from Redis"""
        deleted = shopper_usecase.delete_session(session_id)
        return {"success": True, "session_id": session_id, "deleted": deleted}

    # =========================================================================
    # Protected Admin AI Copilot Chat Endpoint (Requires JWT with ADMIN role)
    # =========================================================================
    @router.post("/chat/admin", response_model=ChatAdminResponse)
    async def chat_admin(
        req: ChatAdminRequest,
        request: Request,
        admin_claims: Dict[str, Any] = Depends(verify_admin_jwt)
    ):
        if not admin_usecase:
            raise HTTPException(status_code=503, detail="Admin AI Copilot service is not initialized.")

        auth_header = request.headers.get("authorization", "")
        token = auth_header[7:].strip() if auth_header.startswith("Bearer ") else ""

        try:
            result = await admin_usecase.chat(
                messages=req.messages,
                admin_claims=admin_claims,
                session_id=req.session_id,
                token=token
            )
            return ChatAdminResponse(
                success=True,
                reply=result.get("reply", ""),
                tool_calls=result.get("tool_calls", []),
                session_id=result.get("session_id"),
                admin_email=result.get("admin_email")
            )
        except Exception as e:
            logger.error(f"ChatAdmin handler exception: {e}", exc_info=True)
            raise HTTPException(status_code=500, detail=str(e))

    return router
