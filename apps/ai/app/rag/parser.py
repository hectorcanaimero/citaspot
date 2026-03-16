# Parser: extrae texto plano de archivos PDF, DOCX y XLSX.
# Llamado por el endpoint /vectorize cuando el payload contiene file_bytes.
from __future__ import annotations

import io
import logging

log = logging.getLogger(__name__)


def extract_text(file_bytes: bytes, file_type: str) -> str:
    """
    Extrae texto plano de un archivo según su tipo.

    Args:
        file_bytes: Contenido binario del archivo.
        file_type:  'pdf', 'docx', 'xlsx' o 'csv'.

    Returns:
        Texto extraído como string plano.

    Raises:
        ValueError: Si el tipo no está soportado o el archivo está corrupto.
    """
    if file_type == "pdf":
        return _extract_pdf(file_bytes)
    elif file_type == "docx":
        return _extract_docx(file_bytes)
    elif file_type in ("xlsx", "xls"):
        return _extract_xlsx(file_bytes)
    elif file_type == "csv":
        return _extract_csv(file_bytes)
    else:
        raise ValueError(f"Tipo de archivo no soportado: {file_type}")


def _extract_pdf(data: bytes) -> str:
    """Extrae texto de un PDF usando pypdf."""
    try:
        from pypdf import PdfReader
    except ImportError as e:
        raise RuntimeError("pypdf no está instalado") from e

    reader = PdfReader(io.BytesIO(data))
    pages: list[str] = []
    for page in reader.pages:
        text = page.extract_text()
        if text:
            pages.append(text.strip())

    result = "\n\n".join(pages)
    if not result.strip():
        raise ValueError("No se pudo extraer texto del PDF. El archivo puede estar escaneado (imagen).")
    log.info("parser.pdf: extraídas %d páginas", len(pages))
    return result


def _extract_docx(data: bytes) -> str:
    """Extrae texto de un archivo DOCX usando python-docx."""
    try:
        from docx import Document
    except ImportError as e:
        raise RuntimeError("python-docx no está instalado") from e

    doc = Document(io.BytesIO(data))
    paragraphs = [p.text.strip() for p in doc.paragraphs if p.text.strip()]

    result = "\n\n".join(paragraphs)
    if not result.strip():
        raise ValueError("No se pudo extraer texto del DOCX. El archivo puede estar vacío.")
    log.info("parser.docx: extraídos %d párrafos", len(paragraphs))
    return result


def _extract_xlsx(data: bytes) -> str:
    """Extrae texto de una hoja de cálculo XLSX usando openpyxl."""
    try:
        import openpyxl
    except ImportError as e:
        raise RuntimeError("openpyxl no está instalado") from e

    wb = openpyxl.load_workbook(io.BytesIO(data), read_only=True, data_only=True)
    rows: list[str] = []

    for sheet in wb.worksheets:
        sheet_rows = list(sheet.rows)
        if not sheet_rows:
            continue

        # Primera fila como encabezado si tiene valores
        header = [str(cell.value).strip() for cell in sheet_rows[0] if cell.value is not None]

        for row in sheet_rows:
            values = [str(cell.value).strip() for cell in row if cell.value is not None]
            if values:
                rows.append(" | ".join(values))

    wb.close()
    result = "\n".join(rows)
    if not result.strip():
        raise ValueError("No se pudo extraer datos de la hoja de cálculo. El archivo puede estar vacío.")
    log.info("parser.xlsx: extraídas %d filas", len(rows))
    return result


def _extract_csv(data: bytes) -> str:
    """Extrae texto de un archivo CSV."""
    import csv

    # Intentar detectar la codificación — primero UTF-8, luego latin-1
    for encoding in ("utf-8-sig", "utf-8", "latin-1"):
        try:
            text = data.decode(encoding)
            break
        except UnicodeDecodeError:
            continue
    else:
        raise ValueError("No se pudo detectar la codificación del CSV.")

    rows: list[str] = []
    reader = csv.reader(text.splitlines())
    for row in reader:
        values = [cell.strip() for cell in row if cell.strip()]
        if values:
            rows.append(" | ".join(values))

    result = "\n".join(rows)
    if not result.strip():
        raise ValueError("No se pudo extraer datos del CSV. El archivo puede estar vacío.")
    log.info("parser.csv: extraídas %d filas", len(rows))
    return result
