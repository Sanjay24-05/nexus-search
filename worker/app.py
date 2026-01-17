from flask import Flask, request, jsonify
import os
from dotenv import load_dotenv

# Load .env file
load_dotenv()

import tempfile
from processor import parse_file, chunk_text
from embeddings import generate_embedding
from storage import save_document, check_quota

app = Flask(__name__)

@app.route('/health')
def health():
    return jsonify({'status': 'ok'}), 200

# Max content length setup if needed, but we do manual check
# app.config['MAX_CONTENT_LENGTH'] = 50 * 1024 * 1024
@app.route('/embed', methods=['POST'])
def embed_text():
    data = request.get_json()
    if not data or 'text' not in data:
        return jsonify({'error': 'No text provided'}), 400
    
    text = data['text']
    try:
        # Generate embedding for the query
        vector = generate_embedding(text)
        return jsonify({'embedding': vector})
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/process', methods=['POST'])
def process_file():
    if 'file' not in request.files:
        return jsonify({'error': 'No file part'}), 400
    
    file = request.files['file']
    user_id = request.form.get('user_id')
    
    if not user_id:
        return jsonify({'error': 'User ID required'}), 400
        
    if file.filename == '':
        return jsonify({'error': 'No selected file'}), 400

    # Check size (approximate from content-length header if avail, or read)
    # Since Go checks header, we assume it's somewhat safe, but strict check needed.
    # We'll save to temp to get size and process.
    
    with tempfile.NamedTemporaryFile(delete=False, suffix="_" + file.filename) as temp:
        file.save(temp)
        temp_path = temp.name
        
    try:
        file_size = os.path.getsize(temp_path)
        
        # 1. Check Quota
        print(f"Checking quota for user {user_id}...")
        if not check_quota(user_id, file_size):
            return jsonify({'error': 'Storage quota exceeded'}), 403
            
        # 2. Parse
        print(f"Parsing file {file.filename}...")
        try:
            text = parse_file(temp_path)
            print(f"File parsed. Length: {len(text)} chars")
        except Exception as e:
             return jsonify({'error': f'Parsing failed: {str(e)}'}), 400

        # 3. Chunk
        chunks = chunk_text(text)
        print(f"Chunked into {len(chunks)} segments. Generating embeddings...")
        
        # 4. Embed
        embeddings = [generate_embedding(chunk) for chunk in chunks]
        print("Embeddings generated.")
        
        # 5. Save
        save_document(user_id, file.filename, chunks, embeddings, file_size)
        print("Document saved to MongoDB.")
        
        return jsonify({'status': 'success', 'chunks': len(chunks), 'size': file_size})
        
    except Exception as e:
        return jsonify({'error': str(e)}), 500
    finally:
        if os.path.exists(temp_path):
            os.remove(temp_path)

if __name__ == '__main__':
    port = int(os.environ.get('PORT', 5000))
    app.run(host='0.0.0.0', port=port)
