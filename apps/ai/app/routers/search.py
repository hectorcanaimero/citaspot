# POST /search — búsqueda semántica en la knowledge base de un tenant.
from __future__ import annotations

from fastapi import APIRouter
from pydantic import BaseModel, Field

from app.rag.store import search as rag_search

router = APIRouter(prefix="/search", tags=["search"])


class SearchRequest(BaseModel):
    tenant_id: str = Field(..., description="UUID del tenant")
    query: str = Field(..., min_length=1, max_length=500, description="Texto de búsqueda")
    top_k: int = Field(default=5, ge=1, le=20)
    min_similarity: float = Field(default=0.70, ge=0.0, le=1.0)


class SearchResult(BaseModel):
    id: str
    document_id: str
    content: str
    similarity: float
    category: str
    title: str


class SearchResponse(BaseModel):
    results: list[SearchResult]
    total: int


@router.post("", response_model=SearchResponse)
async def search(req: SearchRequest) -> SearchResponse:
    """
    Busca los chunks más similares a la query en la knowledge base del tenant.
    Usa cosine similarity sobre embeddings generados con text-embedding-3-small.
    """
    results = await rag_search(
        tenant_id=req.tenant_id,
        query=req.query,
        top_k=req.top_k,
        min_similarity=req.min_similarity,
    )
    items = [SearchResult(**r) for r in results]
    return SearchResponse(results=items, total=len(items))
