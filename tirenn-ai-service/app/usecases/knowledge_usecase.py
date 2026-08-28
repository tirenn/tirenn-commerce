import io
import re
import logging
from typing import List, Dict, Any, Optional

import pypdf
from app.repositories.embedding_repository import EmbeddingRepository
from app.repositories.knowledge_repository import KnowledgeRepository

logger = logging.getLogger("ai-service.usecase.knowledge")

class KnowledgeUseCase:
    """UseCase managing in-memory PDF parsing, text chunking, embedding generation, and vector RAG retrieval"""

    def __init__(
        self,
        knowledge_repo: KnowledgeRepository,
        embedding_repo: EmbeddingRepository
    ):
        self.knowledge_repo = knowledge_repo
        self.embedding_repo = embedding_repo

    def _chunk_text(
        self,
        text: str,
        page_number: int,
        chunk_size: int = 500,
        chunk_overlap: int = 100
    ) -> List[Dict[str, Any]]:
        """Split text into semantically cohesive overlapping chunks without breaking mid-sentence where possible"""
        # Clean redundant whitespaces
        clean_text = re.sub(r'[ \t]+', ' ', text).strip()
        if not clean_text:
            return []

        # Split by paragraphs or double newlines first
        paragraphs = [p.strip() for p in clean_text.split('\n') if p.strip()]
        
        chunks = []
        current_chunk = ""

        for para in paragraphs:
            if len(current_chunk) + len(para) <= chunk_size:
                current_chunk += ("\n" if current_chunk else "") + para
            else:
                if current_chunk:
                    chunks.append(current_chunk)
                # Overlap tail
                overlap_text = current_chunk[-chunk_overlap:] if len(current_chunk) > chunk_overlap else ""
                current_chunk = (overlap_text + "\n" + para).strip() if overlap_text else para

        if current_chunk:
            chunks.append(current_chunk)

        # Fallback if paragraphs were huge single blocks
        final_chunks = []
        for c in chunks:
            if len(c) > chunk_size * 1.5:
                # Sub-split by sentence
                sentences = re.split(r'(?<=[.?!])\s+', c)
                sub_c = ""
                for s in sentences:
                    if len(sub_c) + len(s) <= chunk_size:
                        sub_c += (" " if sub_c else "") + s
                    else:
                        if sub_c:
                            final_chunks.append(sub_c)
                        sub_c = s
                if sub_c:
                    final_chunks.append(sub_c)
            else:
                final_chunks.append(c)

        return [
            {
                "content": fc.strip(),
                "page_number": page_number
            }
            for fc in final_chunks if len(fc.strip()) > 15
        ]

    def index_pdf_in_memory(
        self,
        file_bytes: bytes,
        filename: str,
        title: Optional[str] = None,
        doc_type: str = "GENERAL"
    ) -> Dict[str, Any]:
        """Parse uploaded PDF file entirely in-memory, extract pages, compute vector embeddings, and save to database"""
        logger.info(f"📄 Processing PDF in-memory: filename='{filename}' | size={len(file_bytes)} bytes | doc_type='{doc_type}'")

        # 1. In-memory extraction using pypdf
        pdf_stream = io.BytesIO(file_bytes)
        reader = pypdf.PdfReader(pdf_stream)
        total_pages = len(reader.pages)

        raw_chunks: List[Dict[str, Any]] = []

        for page_idx, page in enumerate(reader.pages):
            page_num = page_idx + 1
            page_text = page.extract_text() or ""
            page_chunks = self._chunk_text(page_text, page_number=page_num)
            raw_chunks.extend(page_chunks)

        if not raw_chunks:
            raise ValueError(f"No readable text could be extracted from PDF '{filename}'. Please ensure the PDF is not an image-only scan.")

        # 2. Assign chunk indices
        for idx, rc in enumerate(raw_chunks):
            rc["chunk_index"] = idx

        # 3. Compute vector embeddings in batch
        texts_to_embed = [c["content"] for c in raw_chunks]
        logger.info(f"🧠 Computing dense embeddings for {len(texts_to_embed)} chunks from '{filename}'...")
        embeddings = self.embedding_repo.encode_batch(texts_to_embed)

        for chunk, emb in zip(raw_chunks, embeddings):
            chunk["embedding"] = emb

        # 4. Save document and vector chunks to database
        doc_title = title or filename.replace(".pdf", "").replace("_", " ").title()
        saved_doc = self.knowledge_repo.save_document_and_chunks(
            title=doc_title,
            doc_type=doc_type.upper(),
            filename=filename,
            total_pages=total_pages,
            chunks=raw_chunks
        )

        logger.info(f"✅ Successfully indexed document ID {saved_doc['id']} ('{doc_title}') with {len(raw_chunks)} vector chunks.")
        return saved_doc

    def list_documents(self) -> List[Dict[str, Any]]:
        """List all indexed knowledge documents"""
        return self.knowledge_repo.list_documents()

    def delete_document(self, doc_id: int) -> bool:
        """Delete an indexed document and its chunks"""
        return self.knowledge_repo.delete_document(doc_id)

    def query_knowledge(
        self,
        query: str,
        limit: int = 5,
        score_threshold: float = 0.15,
        doc_type: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Execute semantic RAG retrieval for user/admin queries"""
        clean_query = query.strip()
        if not clean_query:
            return []

        query_vec = self.embedding_repo.encode(clean_query)
        results = self.knowledge_repo.search_chunks(
            query_vector=query_vec,
            limit=limit,
            score_threshold=score_threshold,
            doc_type=doc_type
        )
        return results
