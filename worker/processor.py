import os
from pypdf import PdfReader
import docx

def parse_file(filepath):
    """
    Parses a file and returns its text content.
    Supports .txt, .pdf, .docx
    """
    ext = os.path.splitext(filepath)[1].lower()
    
    if ext == '.txt':
        with open(filepath, 'r', encoding='utf-8', errors='ignore') as f:
            return f.read()
            
    elif ext == '.pdf':
        reader = PdfReader(filepath)
        text = ""
        for page in reader.pages:
            text += page.extract_text() + "\n"
        return text
        
    elif ext == '.docx':
        doc = docx.Document(filepath)
        text = ""
        for para in doc.paragraphs:
            text += para.text + "\n"
        return text
        
    else:
        raise ValueError(f"Unsupported file type: {ext}")

def chunk_text(text, chunk_size=500):
    """
    Simple chunking strategy.
    """
    words = text.split()
    chunks = []
    current_chunk = []
    current_length = 0
    
    for word in words:
        current_chunk.append(word)
        current_length += len(word) + 1 # +1 for space
        if current_length >= chunk_size:
            chunks.append(" ".join(current_chunk))
            current_chunk = []
            current_length = 0
            
    if current_chunk:
        chunks.append(" ".join(current_chunk))
        
    return chunks
