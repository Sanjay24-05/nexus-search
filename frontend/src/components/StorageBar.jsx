import React, { useEffect, useState } from 'react';

export default function StorageBar({ token, triggerUpdate }) {
    const [usage, setUsage] = useState(0);
    const max = 50 * 1024 * 1024; // 50MB

    const fetchUsage = async () => {
        try {
            const res = await fetch('http://localhost:8080/api/user', {
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (res.ok) {
                const data = await res.json();
                setUsage(data.total_storage_bytes);
            }
        } catch (e) {
            console.error("Failed to fetch storage usage", e);
        }
    };

    useEffect(() => {
        fetchUsage();
    }, [token, triggerUpdate]);

    const percentage = Math.min((usage / max) * 100, 100);
    const color = percentage > 90 ? '#ef4444' : percentage > 70 ? '#eab308' : '#10b981';

    return (
        <div style={{ marginTop: '1rem', padding: '0.5rem', background: 'rgba(0,0,0,0.3)', borderRadius: '0.5rem' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '0.25rem', fontSize: '0.8rem', color: '#cbd5e1' }}>
                <span>Storage Quota</span>
                <span>{(usage / 1024 / 1024).toFixed(2)}MB / 50MB</span>
            </div>
            <div style={{ width: '100%', height: '6px', background: '#334155', borderRadius: '3px', overflow: 'hidden' }}>
                <div style={{ width: `${percentage}%`, height: '100%', background: color, transition: 'width 0.5s' }} />
            </div>
        </div>
    );
}
