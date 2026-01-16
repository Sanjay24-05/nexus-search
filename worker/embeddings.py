from sentence_transformers import SentenceTransformer
import os

# Global model instance
_model = None

def get_model():
    global _model
    if _model is None:
        # Use a small, efficient model for local use
        print("Loading Embedding Model...")
        _model = SentenceTransformer('all-MiniLM-L6-v2')
    return _model

def generate_embedding(text):
    model = get_model()
    return model.encode(text).tolist()
