import logging
from typing import List, Dict, Any, Optional
from pydantic import BaseModel
from fastapi import APIRouter, HTTPException

from app.domain.chat import ChatMessage
from app.usecases.shopper_usecase import ShopperUseCase

logger = logging.getLogger("ai-service.handler.chat")

class ChatShopperRequest(BaseModel):
    messages: List[ChatMessage]
    is_authenticated: Optional[bool] = False
    user_name: Optional[str] = None

class ChatShopperResponse(BaseModel):
    success: bool
    reply: str
    tool_calls: List[Dict[str, Any]] = []
    suggested_products: List[Dict[str, Any]] = []
    cart_action: Optional[Dict[str, Any]] = None

def get_chat_router(shopper_usecase: ShopperUseCase) -> APIRouter:
    """Factory creating chat routes with injected ShopperUseCase"""
    router = APIRouter(tags=["Chat Shopper"])

    async def _handle_chat(req: ChatShopperRequest) -> ChatShopperResponse:
        try:
            result = await shopper_usecase.chat(
                messages=req.messages,
                is_authenticated=req.is_authenticated or False,
                user_name=req.user_name
            )
            return ChatShopperResponse(
                success=True,
                reply=result.reply,
                tool_calls=result.tool_calls,
                suggested_products=result.suggested_products,
                cart_action=result.cart_action
            )
        except Exception as e:
            logger.error(f"ChatShopper handler exception: {e}", exc_info=True)
            raise HTTPException(status_code=500, detail=str(e))

    @router.post("/chat/shopper", response_model=ChatShopperResponse)
    async def chat_shopper(req: ChatShopperRequest):
        return await _handle_chat(req)

    @router.post("/chat", response_model=ChatShopperResponse)
    async def chat(req: ChatShopperRequest):
        return await _handle_chat(req)

    return router
