import React, { useState, useEffect } from 'react';
import Login from './components/Login';
import SearchLayout from './components/SearchLayout';

function App() {
  const [token, setToken] = useState(localStorage.getItem('token'));
  const [user, setUser] = useState(null);

  useEffect(() => {
    if (token) {
       // Optional: Validate token or fetch user details
       // For now, assume valid
    }
  }, [token]);

  const handleLogin = (newToken) => {
    localStorage.setItem('token', newToken);
    setToken(newToken);
  };

  const handleLogout = () => {
    localStorage.removeItem('token');
    setToken(null);
  };

  return (
    <div className="app">
      <header className="glass" style={{ padding: '1rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h1 style={{ margin: 0, background: 'linear-gradient(to right, #60a5fa, #a78bfa)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
          NexusSearch
        </h1>
        {token && (
          <button onClick={handleLogout} className="btn" style={{ background: 'transparent', color: '#cbd5e1' }}>
            Logout
          </button>
        )}
      </header>
      
      <main style={{ flex: 1 }}>
        {!token ? (
          <Login onLogin={handleLogin} />
        ) : (
          <SearchLayout token={token} />
        )}
      </main>
    </div>
  );
}

export default App;
