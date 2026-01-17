import React, { useState } from 'react';

export default function Login({ onLogin }) {
    const [isRegister, setIsRegister] = useState(false);
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');

        // Smart Fallback: If Env Var is missing...
        // 1. If running on localhost, use localhost:8080
        // 2. If running on Vercel/Prod, use the Render URL hardcoded as safety net
        const isLocal = window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1';
        const API_BASE = import.meta.env.VITE_API_URL || (isLocal ? 'http://localhost:8080' : 'https://nexus-search-1.onrender.com');

        console.log("Current API_BASE:", API_BASE);
        const endpoint = isRegister ? `${API_BASE}/api/register` : `${API_BASE}/api/login`;

        try {
            const res = await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password }),
            });

            if (!res.ok) {
                const text = await res.text();
                throw new Error(text || 'Action failed');
            }

            if (!isRegister) {
                // Login returns token logic?
                // Go backend sets HTTP-only cookie.
                // But I also return JSON token? "json.NewEncoder(w).Encode(map[string]string{"token": tokenString})"
                // Yes.
                const data = await res.json();
                if (data.token) {
                    onLogin(data.token);
                }
            } else {
                // After register, switch to login or auto-login
                // Let's just switch to login view
                setIsRegister(false);
                setError('Registration successful! Please login.');
            }
        } catch (err) {
            setError(err.message);
        }
    };

    return (
        <div className="container" style={{ display: 'flex', justifyContent: 'center', marginTop: '4rem' }}>
            <div className="glass" style={{ padding: '2rem', borderRadius: '1rem', width: '100%', maxWidth: '400px' }}>
                <h2 style={{ textAlign: 'center', marginBottom: '2rem' }}>
                    {isRegister ? 'Create Account' : 'Welcome Back'}
                </h2>

                {error && <div style={{ color: '#ef4444', marginBottom: '1rem', textAlign: 'center' }}>{error}</div>}

                <form onSubmit={handleSubmit} style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                    <input
                        type="text"
                        className="input"
                        placeholder="Username"
                        value={username}
                        onChange={(e) => setUsername(e.target.value)}
                        required
                    />
                    <input
                        type="password"
                        className="input"
                        placeholder="Password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        required
                    />
                    <button type="submit" className="btn btn-primary">
                        {isRegister ? 'Sign Up' : 'Login'}
                    </button>
                </form>

                <div style={{ marginTop: '1.5rem', textAlign: 'center', color: '#94a3b8' }}>
                    {isRegister ? "Already have an account? " : "New to Nexus? "}
                    <button
                        onClick={() => setIsRegister(!isRegister)}
                        style={{ background: 'none', border: 'none', color: '#3b82f6', cursor: 'pointer', textDecoration: 'underline' }}
                    >
                        {isRegister ? 'Login' : 'Register'}
                    </button>
                </div>
            </div>
        </div>
    );
}
