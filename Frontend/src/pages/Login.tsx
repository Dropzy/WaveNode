import React, { useEffect, useState } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { useNavigate, Link } from 'react-router-dom';
import { DESKTOP_SERVER_STORAGE_KEY, normalizeServerUrl, setDesktopServerUrl } from '../services/api';
import './Login.css';

interface LoginProps {
  registrationEnabled: boolean;
}

const Login: React.FC<LoginProps> = ({ registrationEnabled }) => {
  const isDesktop = Boolean(window.WAVENODE_DESKTOP);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [serverUrl, setServerUrl] = useState(() => {
    if (!window.WAVENODE_DESKTOP) {
      return '';
    }
    return localStorage.getItem(DESKTOP_SERVER_STORAGE_KEY) || 'http://127.0.0.1:8080';
  });
  const [discoveredServers, setDiscoveredServers] = useState<Array<{ name: string; url: string }>>([]);
  const [isDiscovering, setIsDiscovering] = useState(false);
  const [error, setError] = useState('');
  const { login, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!isDesktop || !window.WAVENODE_DESKTOP_BRIDGE) {
      return;
    }

    let active = true;
    void window.WAVENODE_DESKTOP_BRIDGE.getServerUrl()
      .then(url => {
        if (active && !localStorage.getItem(DESKTOP_SERVER_STORAGE_KEY)) {
          setServerUrl(normalizeServerUrl(url));
        }
      })
      .catch(() => undefined);

    return () => {
      active = false;
    };
  }, [isDesktop]);

  const discoverServers = async () => {
    if (!window.WAVENODE_DESKTOP_BRIDGE) {
      return;
    }

    setIsDiscovering(true);
    setError('');
    try {
      const servers = await window.WAVENODE_DESKTOP_BRIDGE.discoverServers();
      setDiscoveredServers(servers);
      if (servers.length === 0) {
        setError('No WaveNode servers were found on this network. Enter the server address manually.');
      }
    } catch {
      setError('Server discovery failed. Enter the server address manually.');
    } finally {
      setIsDiscovering(false);
    }
  };

  const selectServer = async (url: string) => {
    setServerUrl(url);
    setError('');
    if (window.WAVENODE_DESKTOP_BRIDGE) {
      await window.WAVENODE_DESKTOP_BRIDGE.setServerUrl(url).catch(() => undefined);
    }
    setDesktopServerUrl(url);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    try {
      if (isDesktop) {
        const normalizedServerUrl = setDesktopServerUrl(serverUrl);
        if (window.WAVENODE_DESKTOP_BRIDGE) {
          await window.WAVENODE_DESKTOP_BRIDGE.setServerUrl(normalizedServerUrl);
        }
      }
      await login(username, password);
      navigate('/');
    } catch (err: unknown) {
      const error = err as { response?: { data?: { error?: string } } };
      setError(error.response?.data?.error || 'Login failed. Please try again.');
    }
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h2>Login to WaveNode</h2>
        <form onSubmit={handleSubmit} className="login-form">
          {isDesktop && (
            <div className="desktop-server-panel">
              <div className="form-group">
                <label htmlFor="server-url">WaveNode server</label>
                <input
                  type="url"
                  id="server-url"
                  value={serverUrl}
                  onChange={(e) => setServerUrl(e.target.value)}
                  required
                  disabled={isLoading}
                  placeholder="http://192.168.1.70:8080"
                  autoComplete="url"
                />
              </div>
              <button
                type="button"
                className="secondary-button"
                onClick={() => void discoverServers()}
                disabled={isLoading || isDiscovering}
              >
                {isDiscovering ? 'Searching...' : 'Find servers on this network'}
              </button>
              {discoveredServers.length > 0 && (
                <div className="server-list">
                  {discoveredServers.map(server => (
                    <button
                      type="button"
                      key={server.url}
                      className={normalizeServerUrl(serverUrl) === server.url ? 'server-option selected' : 'server-option'}
                      onClick={() => void selectServer(server.url)}
                    >
                      <strong>{server.name}</strong>
                      <span>{server.url}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
          <div className="form-group">
            <label htmlFor="username">Username</label>
            <input
              type="text"
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
              disabled={isLoading}
              autoComplete="username"
            />
          </div>
          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              type="password"
              id="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
              disabled={isLoading}
              autoComplete="current-password"
            />
          </div>
          {error && <div className="error-message">{error}</div>}
          <button type="submit" disabled={isLoading} className="login-button">
            {isLoading ? 'Logging in...' : 'Login'}
          </button>
        </form>
        {registrationEnabled && (
          <div className="login-footer">
            <p>
              Don't have an account? <Link to="/register">Register here</Link>
            </p>
          </div>
        )}
      </div>
    </div>
  );
};

export default Login;
