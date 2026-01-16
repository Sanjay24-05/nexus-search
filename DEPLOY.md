# Deployment Guide for NexusSearch

## Why "Render + Vercel"?
**GitHub Pages** is for static websites only. It cannot run your Go Backend or Python Worker.
For a full-stack app like this, the best free/cheap combination is:
1.  **Render**: Hosts the **Go Gateway** and **Python Worker** (Docker/Web Services).
2.  **Vercel**: Hosts the **React Frontend** (Fastest static hosting).

---

## Part 1: Databases (MongoDB Atlas)
You are already using MongoDB Atlas.
1.  Ensure "Network Access" in Atlas is set to allow access from **Anywhere (0.0.0.0/0)** (since Render IPs change).

---

## Part 2: Backend & Worker (Render)
Create an account at [render.com](https://render.com).

### 1. Deploy Python Worker
1.  Click **New +** -> **Web Service**.
2.  Connect your `nexus-search` GitHub repo.
3.  **Root Directory**: `worker`
4.  **Runtime**: Docker (Recommended) OR Python 3.
    *   If using Python runtime: Build Command: `pip install -r requirements.txt`, Start Command: `python app.py`.
5.  **Environment Variables**:
    *   `MONGODB_URI`: (Your Atlas connection string)
    *   `PORT`: `5000` (Render will auto-set this, just ensure your code reads `os.getenv('PORT')`)
6.  Click **Create Web Service**.
7.  **Copy the URL** provided by Render (e.g., `https://nexus-worker.onrender.com`).

### 2. Deploy Go Gateway
1.  Click **New +** -> **Web Service**.
2.  Connect your `nexus-search` GitHub repo.
3.  **Root Directory**: `gateway`
4.  **Runtime**: Docker.
5.  **Environment Variables**:
    *   `MONGODB_URI`: (Your Atlas connection string)
    *   `JWT_SECRET`: (Generate a random string)
    *   `SERPAPI_KEY`: (Your SerpApi Key)
    *   `WORKER_URL`: **The URL from Step 1** (e.g., `https://nexus-worker.onrender.com`)
    *   `ALLOWED_ORIGINS`: `https://your-vercel-frontend-url.vercel.app` (You will get this in Part 3. For now, put `*` to test).
6.  Click **Create Web Service**.
7.  **Copy the URL** provided by Render (e.g., `https://nexus-gateway.onrender.com`).

---

## Part 3: Frontend (Vercel)
Create an account at [vercel.com](https://vercel.com).

1.  Click **Add New...** -> **Project**.
2.  Import `nexus-search` repo.
3.  **Root Directory**: Edit this -> select `frontend`.
4.  **Build Settings**: Default (`vite`, `npm run build`, `dist`) is fine.
5.  **Environment Variables**:
    *   The frontend code currently points to `localhost`. We need to change this!
    *   **Action**: You need to update `SearchLayout.jsx` to use an environment variable for the API URL.

### 3a. Update Frontend Code (Before Deploying)
I have updated your `SearchLayout.jsx` to read `import.meta.env.VITE_API_URL`.

**On Vercel Dashboard:**
    *   Add Variable: `VITE_API_URL`
    *   Value: **The Gateway URL from Part 2** (e.g., `https://nexus-gateway.onrender.com`) - **IMPORTANT**: No trailing slash.

6.  Click **Deploy**.

---

## Final Step
Once Vercel gives you your live URL (e.g., `https://nexus-search-alpha.vercel.app`), go back to **Render (Go Gateway)** and update:
*   `ALLOWED_ORIGINS`: `https://nexus-search-alpha.vercel.app`
