import logging
from typing import Optional, List, Dict, Any
from fastapi import APIRouter, UploadFile, File, Form, Depends, HTTPException, status
from pydantic import BaseModel

from app.core.security import verify_admin_jwt
from app.usecases.knowledge_usecase import KnowledgeUseCase

logger = logging.getLogger("ai-service.handler.knowledge")

class KnowledgeQueryRequest(BaseModel):
    query: str
    limit: Optional[int] = 5
    score_threshold: Optional[float] = 0.15
    doc_type: Optional[str] = None

class KnowledgeQueryResponse(BaseModel):
    success: bool
    query: str
    total_results: int
    results: List[Dict[str, Any]]

def get_knowledge_router(knowledge_usecase: KnowledgeUseCase) -> APIRouter:
    router = APIRouter(prefix="/knowledge", tags=["Knowledge Base & Vector RAG"])

    @router.post("/upload-pdf")
    async def upload_pdf(
        file: UploadFile = File(...),
        title: Optional[str] = Form(None),
        doc_type: Optional[str] = Form("GENERAL"),
        claims: Dict[str, Any] = Depends(verify_admin_jwt)
    ):
        """Upload a PDF file, parse entirely in-memory, chunk, and index into pgvector (Admin Only)"""
        if not file.filename.lower().endswith(".pdf"):
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="Only PDF files (.pdf) are supported for knowledge indexing."
            )

        try:
            # Read file bytes in-memory (never write to disk)
            file_bytes = await file.read()
            if len(file_bytes) == 0:
                raise HTTPException(
                    status_code=status.HTTP_400_BAD_REQUEST,
                    detail="Uploaded PDF file is empty."
                )

            logger.info(f"Admin '{claims.get('email')}' uploaded PDF '{file.filename}' ({len(file_bytes)} bytes)")
            
            doc_record = knowledge_usecase.index_pdf_in_memory(
                file_bytes=file_bytes,
                filename=file.filename,
                title=title,
                doc_type=doc_type or "GENERAL"
            )

            return {
                "success": True,
                "message": f"PDF '{file.filename}' was successfully vectorized and indexed into RAG Knowledge Base.",
                "document": doc_record
            }

        except ValueError as ve:
            logger.warning(f"PDF extraction error: {ve}")
            raise HTTPException(status_code=status.HTTP_422_UNPROCESSABLE_ENTITY, detail=str(ve))
        except Exception as e:
            logger.error(f"Failed to process PDF upload: {e}", exc_info=True)
            raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=f"PDF processing failed: {str(e)}")

    @router.get("/documents")
    async def list_documents(claims: Dict[str, Any] = Depends(verify_admin_jwt)):
        """List all indexed knowledge documents with chunk counts (Admin Only)"""
        try:
            docs = knowledge_usecase.list_documents()
            return {
                "success": True,
                "total": len(docs),
                "documents": docs
            }
        except Exception as e:
            logger.error(f"Failed to list documents: {e}", exc_info=True)
            raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e))

    @router.delete("/documents/{doc_id}")
    async def delete_document(doc_id: int, claims: Dict[str, Any] = Depends(verify_admin_jwt)):
        """Delete an indexed document and its vector chunks (Admin Only)"""
        try:
            deleted = knowledge_usecase.delete_document(doc_id)
            if not deleted:
                raise HTTPException(status_code=status.HTTP_404_NOT_FOUND, detail=f"Document ID {doc_id} not found.")
            return {
                "success": True,
                "message": f"Document ID {doc_id} and all associated vector chunks were deleted."
            }
        except HTTPException:
            raise
        except Exception as e:
            logger.error(f"Failed to delete document: {e}", exc_info=True)
            raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e))

    @router.post("/query", response_model=KnowledgeQueryResponse)
    async def query_knowledge(req: KnowledgeQueryRequest):
        """Semantic Vector Search across RAG Knowledge Chunks"""
        try:
            results = knowledge_usecase.query_knowledge(
                query=req.query,
                limit=req.limit or 5,
                score_threshold=req.score_threshold or 0.15,
                doc_type=req.doc_type
            )
            return KnowledgeQueryResponse(
                success=True,
                query=req.query,
                total_results=len(results),
                results=results
            )
        except Exception as e:
            logger.error(f"Knowledge query error: {e}", exc_info=True)
            raise HTTPException(status_code=status.HTTP_500_INTERNAL_SERVER_ERROR, detail=str(e))

    return router
