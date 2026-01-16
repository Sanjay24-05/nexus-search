import React, { useState } from 'react';
import StorageBar from './StorageBar';

export default function SearchLayout({ token }) {
    const [query, setQuery] = useState('');
    const [results, setResults] = useState(null);
    const [loading, setLoading] = useState(false);
    const [isPKB, setIsPKB] = useState(false); // Toggle
    const [activeToggles, setToggles] = useState({ web: true, wiki: true, ddg: true });
    const [uploadCount, setUploadCount] = useState(0);

    // Get API base URL from env, default to localhost for dev
    const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

    const handleSearch = async (e) => {
        e.preventDefault();
        if (!query.trim()) return;

        setLoading(true);
        try {
            // Params: q, web=true, wiki=true...
            const params = new URLSearchParams({
                q: query,
                web: activeToggles.web,
                wiki: activeToggles.wiki,
                ddg: activeToggles.ddg,
                pkb: isPKB
            });

            const res = await fetch(`${API_BASE}/api/search?${params.toString()}`, {
                headers: { 'Authorization': `Bearer ${token}` }
            });

            if (!res.ok) throw new Error('Search failed');
            const data = await res.json();
            setResults(data.results);

        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    const handleUpload = async (e) => {
        const file = e.target.files[0];
        if (!file) return;

        // Auto-enable PKB on upload
        setIsPKB(true);

        const formData = new FormData();
        formData.append('file', file);

        try {
            const res = await fetch(`${API_BASE}/api/upload`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`
                    // Content-Type header skips for FormData to allow boundary
                },
                body: formData
            });

            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || 'Upload failed');
            }

            alert(`File ${file.name} uploaded successfully!`);
            setUploadCount(prev => prev + 1); // Trigger storage update
        } catch (err) {
            console.error(err);
            alert(`Upload failed: ${err.message}`);
        }
    };

    return (
        <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            {/* Controls Bar */}
            <div className="glass" style={{ padding: '1rem', margin: '1rem', borderRadius: '1rem', display: 'flex', gap: '1rem', alignItems: 'center', flexWrap: 'wrap' }}>
                <form onSubmit={handleSearch} style={{ flex: 1, display: 'flex', gap: '0.5rem' }}>
                    <input
                        className="input"
                        value={query}
                        onChange={e => setQuery(e.target.value)}
                        placeholder="Ask anything..."
                    />
                    <button type="submit" className="btn btn-primary" disabled={loading}>
                        {loading ? '...' : 'Search'}
                    </button>
                </form>

                <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', fontSize: '0.9rem' }}>
                        <input type="checkbox" checked={activeToggles.web} onChange={e => setToggles({ ...activeToggles, web: e.target.checked })} /> Google
                    </label>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', fontSize: '0.9rem' }}>
                        <input type="checkbox" checked={activeToggles.ddg} onChange={e => setToggles({ ...activeToggles, ddg: e.target.checked })} /> DDG
                    </label>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '0.25rem', fontSize: '0.9rem' }}>
                        <input type="checkbox" checked={activeToggles.wiki} onChange={e => setToggles({ ...activeToggles, wiki: e.target.checked })} /> Wiki
                    </label>
                </div>

                <div style={{ borderLeft: '1px solid var(--secondary-color)', paddingLeft: '1rem', display: 'flex', gap: '1rem', alignItems: 'center' }}>
                    <label className="btn" style={{ background: isPKB ? 'var(--primary-color)' : 'var(--card-bg)', fontSize: '0.9rem' }}>
                        PKB Mode
                        <input type="checkbox" checked={isPKB} onChange={e => setIsPKB(e.target.checked)} style={{ marginLeft: '0.5rem' }} />
                    </label>

                    <label className="btn" style={{ background: 'var(--card-bg)', border: '1px solid var(--primary-color)', color: 'var(--primary-color)' }}>
                        Upload Doc
                        <input type="file" onChange={handleUpload} style={{ display: 'none' }} />
                    </label>
                    <StorageBar token={token} triggerUpdate={uploadCount} />
                </div>
            </div>

            {/* Results Area */}
            <div className={`split-pane`}>
                {/* Left Pane: Web Results */}
                <div className={`pane glass ${isPKB ? 'left' : 'full'}`}>
                    <h3 style={{ marginTop: 0, color: 'var(--primary-color)' }}>Web Results</h3>
                    {results ? results.filter(r => !r.source.startsWith('PKB')).map((r, i) => (
                        <div key={i} style={{ marginBottom: '1.5rem' }}>
                            <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>{r.source}</div>
                            <a href={r.url} target="_blank" rel="noreferrer" style={{ fontSize: '1.1rem', color: '#60a5fa', textDecoration: 'none', fontWeight: 'bold' }}>{r.title}</a>
                            <p style={{ marginTop: '0.25rem', color: '#cbd5e1' }}>{r.snippet}</p>
                        </div>
                    )) : <div style={{ color: '#64748b' }}>Search results will appear here...</div>}
                </div>

                {/* Right Pane: PKB (Only if isPKB) */}
                {isPKB && (
                    <div className="pane glass">
                        <h3 style={{ marginTop: 0, color: 'var(--success)' }}>Personal Knowledge Base</h3>
                        {results ? results.filter(r => r.source.startsWith('PKB')).map((r, i) => (
                            <div key={i} style={{ marginBottom: '1rem', padding: '1rem', background: 'rgba(0,0,0,0.2)', borderRadius: '0.5rem', border: '1px solid rgba(255,255,255,0.05)' }}>
                                <div style={{ fontWeight: 'bold', color: 'var(--success)', marginBottom: '0.4rem' }}>{r.title}</div>
                                <div style={{
                                    fontSize: '0.85rem',
                                    opacity: 0.8,
                                    display: '-webkit-box',
                                    WebkitLineClamp: 4,
                                    WebkitBoxOrient: 'vertical',
                                    overflow: 'hidden',
                                    textOverflow: 'ellipsis'
                                }}>
                                    {r.snippet}
                                </div>
                            </div>
                        )) : <div style={{ color: '#64748b' }}>Local documents will appear here...</div>}

                        {(!results || results.filter(r => r.source.startsWith('PKB')).length === 0) && (
                            <div style={{ marginTop: '2rem', textAlign: 'center', opacity: 0.5 }}>
                                No relevant documents found.
                            </div>
                        )}
                    </div>
                )}
            </div>
        </div >
    );
}
