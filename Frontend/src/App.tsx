import { useEffect, useState } from 'react';
import { Navigate, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { AudioProvider } from './contexts/AudioContext';
import ProtectedRoute from './components/ProtectedRoute';
import { Layout } from './components/Layout';
import { Home } from './pages/Home';
import { Search } from './pages/Search';
import { Library } from './pages/Library';
import { LikedSongs } from './pages/LikedSongs';
import { PlaylistPage } from './pages/Playlist';
import { Album } from './pages/Album';
import { Artist } from './pages/Artist';
import AdminDashboard from './pages/AdminDashboard';
import Login from './pages/Login';
import Register from './pages/Register';
import Setup from './pages/Setup';
import Account from './pages/Account';
import SmartPlaylistEditor from './pages/SmartPlaylistEditor';
import History from './pages/History';
import LastFMCallback from './pages/LastFMCallback';
import { GlobalStyle } from './styles/GlobalStyle';
import { AppThemeProvider } from './contexts/ThemeContext';
import { setupAPI, SetupStatus } from './services/api';
import { Loader2 } from 'lucide-react';
import styled from 'styled-components';

function SetupAwareApp() {
  const [status, setStatus] = useState<SetupStatus | null>(null)
  const [error, setError] = useState('')

  const loadStatus = async () => {
    setError('')
    try {
      setStatus(await setupAPI.getStatus())
    } catch {
      setError('WaveNode could not check whether setup is complete.')
    }
  }

  useEffect(() => {
    void loadStatus()
  }, [])

  if (error) {
    return (
      <GateState>
        <h1>WaveNode is unavailable</h1>
        <p>{error}</p>
        <button type="button" onClick={() => void loadStatus()}>Try again</button>
      </GateState>
    )
  }

  if (!status) {
    return <GateState><Loader2 className="spin" size={30} /><p>Starting WaveNode...</p></GateState>
  }

  if (status.required) {
    return (
      <Routes>
        <Route path="/setup" element={<Setup status={status} />} />
        <Route path="*" element={<Navigate to="/setup" replace />} />
      </Routes>
    )
  }

  return (
    <AudioProvider>
      <Routes>
        <Route path="/setup" element={<Navigate to="/" replace />} />
        <Route path="/login" element={<Login registrationEnabled={status.registration_enabled} />} />
        <Route
          path="/register"
          element={status.registration_enabled ? <Register /> : <Navigate to="/login" replace />}
        />
        <Route path="/" element={<ProtectedRoute><Layout><Home /></Layout></ProtectedRoute>} />
        <Route path="/search" element={<ProtectedRoute><Layout><Search /></Layout></ProtectedRoute>} />
        <Route path="/library" element={<ProtectedRoute><Layout><Library /></Layout></ProtectedRoute>} />
        <Route path="/liked-songs" element={<ProtectedRoute><Layout><LikedSongs /></Layout></ProtectedRoute>} />
        <Route path="/album/:albumName" element={<ProtectedRoute><Layout><Album /></Layout></ProtectedRoute>} />
        <Route path="/artist/:artistId" element={<ProtectedRoute><Layout><Artist /></Layout></ProtectedRoute>} />
        <Route path="/playlist/:id" element={<ProtectedRoute><Layout><PlaylistPage /></Layout></ProtectedRoute>} />
        <Route path="/smart-playlist/new" element={<ProtectedRoute><Layout><SmartPlaylistEditor /></Layout></ProtectedRoute>} />
        <Route path="/smart-playlist/:id/edit" element={<ProtectedRoute><Layout><SmartPlaylistEditor /></Layout></ProtectedRoute>} />
        <Route path="/admin" element={<ProtectedRoute><Layout><AdminDashboard /></Layout></ProtectedRoute>} />
        <Route path="/account" element={<ProtectedRoute><Layout><Account /></Layout></ProtectedRoute>} />
        <Route path="/lastfm/callback" element={<ProtectedRoute><Layout><LastFMCallback /></Layout></ProtectedRoute>} />
        <Route path="/history" element={<ProtectedRoute><Layout><History /></Layout></ProtectedRoute>} />
      </Routes>
    </AudioProvider>
  )
}

function App() {
  return (
    <AppThemeProvider>
      <GlobalStyle />
      <AuthProvider>
        <SetupAwareApp />
      </AuthProvider>
    </AppThemeProvider>
  );
}

const GateState = styled.main`
  min-height: 100vh;
  display: grid;
  place-content: center;
  justify-items: center;
  gap: 14px;
  background: ${({ theme }) => theme.colors.background};
  color: ${({ theme }) => theme.colors.text};

  p { color: ${({ theme }) => theme.colors.muted}; }
  button {
    padding: 11px 18px;
    border-radius: 999px;
    background: ${({ theme }) => theme.colors.accentGradient};
    color: ${({ theme }) => theme.colors.accentText};
    font-weight: 800;
  }
`

export default App;
