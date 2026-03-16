-- 001_extensions.sql
-- Habilitar extensiones necesarias para CitaSpot

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";   -- Generación de UUIDs
CREATE EXTENSION IF NOT EXISTS "vector";       -- pgvector para RAG
CREATE EXTENSION IF NOT EXISTS "pg_trgm";      -- Búsqueda de texto (GIN index)
