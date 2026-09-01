import asyncio
import os
import psycopg2
import httpx
import numpy as np
from dotenv import load_dotenv

load_dotenv()

DB_HOST = os.getenv("DB_HOST", "127.0.0.1")
DB_PORT = os.getenv("DB_PORT", "5432")
DB_USER = os.getenv("DB_USER", "postgres_user111")
DB_PASSWORD = os.getenv("DB_PASSWORD", "password123!!!")
DB_NAME = os.getenv("DB_NAME", "commerce_db")
OLLAMA_BASE_URL = os.getenv("OLLAMA_BASE_URL", "http://127.0.0.1:11434").rstrip("/")
EMBEDDING_MODEL_NAME = os.getenv("EMBEDDING_MODEL_NAME", "bge-m3")

def normalize(vec):
    arr = np.array(vec, dtype=np.float32)
    norm = np.linalg.norm(arr)
    if norm > 0:
        arr = arr / norm
    return arr.tolist()

def get_embedding(client, text):
    try:
        resp = client.post(f"{OLLAMA_BASE_URL}/api/embed", json={"model": EMBEDDING_MODEL_NAME, "input": text}, timeout=30.0)
        if resp.status_code == 200:
            embeddings = resp.json().get("embeddings", [])
            if embeddings:
                return normalize(embeddings[0])
    except Exception as e:
        print(f"Error fetching embedding: {e}")
    return None

def main():
    print(f"Connecting to PostgreSQL at {DB_HOST}:{DB_PORT}/{DB_NAME}...")
    conn = psycopg2.connect(
        host=DB_HOST,
        port=DB_PORT,
        user=DB_USER,
        password=DB_PASSWORD,
        dbname=DB_NAME
    )
    cursor = conn.cursor()

    cursor.execute("SELECT id, name, description, badge FROM products ORDER BY id ASC;")
    products = cursor.fetchall()
    print(f"Found {len(products)} products to compute embeddings for with Ollama ({EMBEDDING_MODEL_NAME})...")

    with httpx.Client() as client:
        for idx, (p_id, name, desc, badge) in enumerate(products):
            text_to_embed = f"{name}. {desc}"
            if badge:
                text_to_embed += f" Badge: {badge}"

            emb = get_embedding(client, text_to_embed)
            if emb:
                vector_str = "[" + ",".join(str(x) for x in emb) + "]"
                cursor.execute("UPDATE products SET embedding = %s WHERE id = %s", (vector_str, p_id))
                conn.commit()
                print(f"  [{idx+1}/{len(products)}] Embedded product #{p_id}: {name[:40]}... ({len(emb)} dim)")
            else:
                print(f"  Failed embedding for product #{p_id}")

    cursor.close()
    conn.close()
    print("All product embeddings updated successfully in PostgreSQL!")

if __name__ == "__main__":
    main()
