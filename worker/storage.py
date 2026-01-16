import os
import pymongo
from bson.objectid import ObjectId

MONGO_URI = os.environ.get("MONGODB_URI")
if not MONGO_URI:
    # Fallback or error
    pass 

client = pymongo.MongoClient(MONGO_URI)
db = client["nexus_search"]
users_col = db["users"]
docs_col = db["docs"]

def get_user_storage(user_id):
    user = users_col.find_one({"_id": ObjectId(user_id)})
    if user:
        return user.get("total_storage_bytes", 0)
    return 0

def update_user_storage(user_id, delta_bytes):
    users_col.update_one(
        {"_id": ObjectId(user_id)},
        {"$inc": {"total_storage_bytes": delta_bytes}}
    )

def check_quota(user_id, new_file_size):
    current = get_user_storage(user_id)
    if current + new_file_size > 50 * 1024 * 1024: # 50MB
        return False
    return True

def save_document(user_id, filename, chunks, embeddings, file_size):
    # Transactional logic ideally, but simple here
    
    # Update quota first? Or after? 
    # Prompt says "Python must verify total storage usage in MongoDB before committing new files."
    
    if not check_quota(user_id, file_size):
        raise Exception("Storage quota exceeded")
    
    update_user_storage(user_id, file_size)
    
    # Save chunks/doc
    # Storing chunks individually or as one doc? 
    # "Docs Collection: Store chunks and embeddings."
    # We'll store one document per file, with chunks array? 
    # Or multiple documents? Vector search works best on chunks.
    # We will insert MULTIPLE documents into 'docs_col', one per chunk.
    # Metadata repeated? Or parent doc?
    # Let's simple: Each chunk is a doc in 'docs' collection.
    
    docs = []
    for i, (chunk_text, vector) in enumerate(zip(chunks, embeddings)):
        doc = {
            "user_id": ObjectId(user_id),
            "filename": filename,
            "chunk_index": i,
            "content": chunk_text,
            "embedding": vector,
            "size_bytes": file_size, # Might be misleading if summed. 
            # We track TOTAL storage on USER entity. 
            # This 'doc' is just a search unit.
            "created_at": "now" # TODO: timestamp
        }
        docs.append(doc)
        
    docs_col.insert_many(docs)
