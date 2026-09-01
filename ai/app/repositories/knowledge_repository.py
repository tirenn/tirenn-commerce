import logging
from typing import List, Optional, Dict, Any
import psycopg2
from psycopg2.extras import RealDictCursor
from app.core.config import settings

logger = logging.getLogger("ai-service.repository.knowledge")

class KnowledgeRepository:
    """Repository handling all PostgreSQL pgvector storage for RAG Knowledge Documents and Chunks"""

    def __init__(self):
        self._init_db()

    def _get_connection(self):
        """Establish PostgreSQL database connection"""
        return psycopg2.connect(
            host=settings.DB_HOST,
            port=settings.DB_PORT,
            user=settings.DB_USER,
            password=settings.DB_PASSWORD,
            dbname=settings.DB_NAME,
            connect_timeout=5
        )

    def _init_db(self):
        """Verify database connectivity (Schema DDL and migrations are managed by Golang backend)"""
        try:
            with self._get_connection() as conn:
                with conn.cursor() as cur:
                    cur.execute("SELECT 1;")
            logger.info("🐘 PostgreSQL pgvector Knowledge Base connection verified (tables managed by Golang backend).")
        except Exception as e:
            logger.warning(f"Knowledge Base DB connection check warning: {e}")

    def _to_vector_str(self, vec: List[float]) -> str:
        """Format float list into PostgreSQL vector literal '[0.12,0.34,...]'"""
        return "[" + ",".join(f"{x:.6f}" for x in vec) + "]"

    def save_document_and_chunks(
        self,
        title: str,
        doc_type: str,
        filename: str,
        total_pages: int,
        chunks: List[Dict[str, Any]]
    ) -> Dict[str, Any]:
        """Save a new knowledge document and its vector chunks in an atomic transaction"""
        with self._get_connection() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute("""
                    INSERT INTO knowledge_documents (title, doc_type, filename, total_pages, total_chunks)
                    VALUES (%s, %s, %s, %s, %s)
                    RETURNING id, title, doc_type, filename, total_pages, total_chunks, created_at;
                """, (title, doc_type, filename, total_pages, len(chunks)))
                doc_record = cur.fetchone()
                doc_id = doc_record["id"]

                # Batch insert chunks
                for chunk in chunks:
                    vec_str = self._to_vector_str(chunk["embedding"]) if chunk.get("embedding") else None
                    cur.execute("""
                        INSERT INTO knowledge_chunks (document_id, chunk_index, content, page_number, embedding)
                        VALUES (%s, %s, %s, %s, %s::vector);
                    """, (
                        doc_id,
                        chunk["chunk_index"],
                        chunk["content"],
                        chunk.get("page_number", 1),
                        vec_str
                    ))

                conn.commit()
                return dict(doc_record)

    def list_documents(self) -> List[Dict[str, Any]]:
        """List all indexed knowledge documents with chunk counts"""
        with self._get_connection() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute("""
                    SELECT id, title, doc_type, filename, total_pages, total_chunks, created_at
                    FROM knowledge_documents
                    ORDER BY id DESC;
                """)
                rows = cur.fetchall()
                return [dict(r) for r in rows]

    def delete_document(self, doc_id: int) -> bool:
        """Delete a document and all its associated vector chunks (cascade)"""
        with self._get_connection() as conn:
            with conn.cursor() as cur:
                cur.execute("DELETE FROM knowledge_documents WHERE id = %s;", (doc_id,))
                deleted = cur.rowcount > 0
                conn.commit()
                return deleted

    def search_chunks(
        self,
        query_vector: List[float],
        limit: int = 5,
        score_threshold: float = 0.15,
        doc_type: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """Semantic Vector Search across knowledge chunks using cosine distance (<=>)"""
        vec_str = self._to_vector_str(query_vector)

        filter_clause = ""
        params: List[Any] = [vec_str, score_threshold]

        if doc_type and doc_type != "ALL":
            filter_clause = "AND d.doc_type = %s"
            params.append(doc_type)

        params.append(limit)

        sql = f"""
            SELECT 
                c.id AS chunk_id,
                c.document_id,
                d.title AS document_title,
                d.doc_type,
                d.filename,
                c.chunk_index,
                c.page_number,
                c.content,
                (1.0 - (c.embedding <=> %s::vector)) AS score
            FROM knowledge_chunks c
            JOIN knowledge_documents d ON c.document_id = d.id
            WHERE c.embedding IS NOT NULL
              AND (1.0 - (c.embedding <=> %s::vector)) >= %s
              {filter_clause}
            ORDER BY score DESC
            LIMIT %s;
        """

        with self._get_connection() as conn:
            with conn.cursor(cursor_factory=RealDictCursor) as cur:
                cur.execute(sql, (vec_str, vec_str, score_threshold, *( [doc_type] if doc_type and doc_type != "ALL" else [] ), limit))
                rows = cur.fetchall()
                return [dict(r) for r in rows]
