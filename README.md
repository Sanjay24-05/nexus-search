# NexusSearch 🚀

NexusSearch is a high-performance **Hybrid Search Engine** that seamlessly merges live Web Search results with a private **Personal Knowledge Base (PKB)**. It allows users to query the internet and their own uploaded documents (PDF, DOCX, TXT) in a single, unified, split-pane interface.

## 🌟 Key Features

-   **Hybrid Orchestration**: Parallel search across Google (SerpApi), DuckDuckGo, Wikipedia, and your private documents.
-   **Vector-Powered PKB**: Uses `all-MiniLM-L6-v2` sentence embeddings for high-accuracy semantic search.
-   **Performance Optimized**: Features batch embedding processing and pre-downloaded ML models for sub-second responses even on cloud free-tiers.
-   **Security First**: JWT-based authentication, secure CORS handling, and strictly isolated user data.
-   **Storage Management**: Real-time tracking of a 50MB storage quota per user.
-   **Responsive Design**: A premium, dark-themed UI built for clarity and speed.

---

## 🏗️ Architecture

NexusSearch follow a modern microservices architecture designed for scalability and separation of concerns:

### 1. React Frontend (Vercel)
-   Built with **Vite** and **React**.
-   Features a dynamic split-pane layout and "Smart API URL" detection for seamless local-to-production transitions.

### 2. Go Gateway (Render)
-   The central nervous system of the app.
-   Handles user authentication, rate limiting, and parallel search orchestration.
-   **Tech**: Go, `rs/cors`, `golang-jwt`, `mongodb-go-driver`.

### 3. Python Worker (Render)
-   Dedicated ML worker for parsing and embedding generation.
-   Uses **Flask** to expose endpoints for document processing and query vectorization.
-   **Tech**: Python, `sentence-transformers`, `pypdf`, `python-docx`.

### 4. Database (MongoDB Atlas)
-   Stores user profiles, document metadata, and high-dimensional vectors.
-   Utilizes **Atlas Vector Search** for semantic similarity matching.

---

## 🎨 Design Choices & Optimizations

-   **Batch Embedding**: Instead of sequential processing, the Python worker processes document chunks in parallel batches, reducing upload latency by ~80%.
-   **Model "Baking"**: To avoid "Cold Start" timeouts on Render, the embedding models are pre-downloaded into the Docker image during the build phase.
-   **Smart Fallback**: The frontend code dynamically switches between your production Render URL and `localhost:8080`, allowing for zero-config development.
-   **CORS Sanitation**: The API gateway implements a robust "Trim & sanitize" middleware to prevent origin mismatches caused by trailing slashes or whitespace.

---

## 🚀 Setup & Deployment

### Local Development
1.  **Gateway**: `cd gateway && go run main.go` (Requires `.env` with `MONGODB_URI`, `JWT_SECRET`, and `SERPAPI_KEY`)
2.  **Worker**: `cd worker && pip install -r requirements.txt && python app.py`
3.  **Frontend**: `cd frontend && npm install && npm run dev`

### Production Configuration
-   **Go Gateway Env Vars**:
    -   `WORKER_URL`: Your Render worker URL (no trailing slash).
    -   `ALLOWED_ORIGINS`: `*` (or your Vercel URL).
-   **Frontend Env Vars**:
    -   `VITE_API_URL`: Your Render gateway URL.

---

## 🔍 Troubleshooting (Free Tier Tips)

Since the backend is hosted on Render's Free Tier:
-   **The 502/404 Error**: If the app hasn't been used for 15 minutes, it goes to "sleep". If you see an error, refresh the page and wait ~30 seconds for the services to wake up.
-   **Pre-Warming**: Opening the Gateway URL directly in a browser tab is the fastest way to wake it up manually.

---

