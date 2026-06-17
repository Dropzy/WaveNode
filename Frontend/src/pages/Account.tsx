import { useEffect, useRef, useState } from 'react'
import styled from 'styled-components'
import { Headphones, KeyRound, LogOut, MonitorSmartphone, Palette, Radio, RefreshCw, UserRound, XCircle } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { accountAPI, api, type PlaybackProfile, type ScrobbleSettings, type UserSession } from '../services/api'
import { useAppTheme } from '../contexts/ThemeContext'
import { appThemes, type AppThemeName } from '../styles/themes'

const Container = styled.section`
  max-width: 760px;
  padding: 32px;
  overflow-y: auto;

  @media (max-width: 768px) {
    padding: 80px 16px 32px;
  }
`

const Card = styled.section`
  margin-top: 22px;
  padding: 22px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 12px;
  background: ${({ theme }) => theme.colors.surfaceSoft};
  box-shadow: 0 18px 45px ${({ theme }) => theme.colors.shadow};
`

const Form = styled.form`
  display: grid;
  gap: 14px;
  margin-top: 18px;

  label {
    display: grid;
    gap: 7px;
    color: ${({ theme }) => theme.colors.muted};
    font-size: 14px;
  }

  input {
    padding: 11px 12px;
    border: 1px solid ${({ theme }) => theme.colors.border};
    border-radius: 8px;
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.surface};
  }
`

const Button = styled.button<{ $danger?: boolean }>`
  width: fit-content;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 999px;
  color: ${props => props.$danger ? '#fff' : props.theme.colors.accentText};
  background: ${props => props.$danger ? props.theme.colors.danger : props.theme.colors.accentGradient};
  font-weight: 800;
`

const Message = styled.p<{ $error?: boolean }>`
  margin-top: 12px;
  color: ${props => props.$error ? props.theme.colors.danger : props.theme.colors.success};
`

const SettingsGrid = styled.div`display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin-top:18px;@media(max-width:640px){grid-template-columns:1fr;}`
const Setting = styled.label`display:grid;gap:7px;color:${props => props.theme.colors.muted};font-size:14px;select,input{padding:11px 12px;border:1px solid ${props => props.theme.colors.border};border-radius:8px;color:${props => props.theme.colors.text};background:${props => props.theme.colors.surface};}`
const CheckSetting = styled.label`display:flex;align-items:center;gap:10px;margin:16px 0;color:${props => props.theme.colors.muted};input{accent-color:${props => props.theme.colors.accent};}`
const SessionList = styled.div`display:grid;gap:9px;margin:16px 0;`
const SessionRow = styled.div`display:flex;justify-content:space-between;align-items:center;gap:15px;padding:13px;border:1px solid ${props => props.theme.colors.border};border-radius:9px;background:${props => props.theme.colors.surface};div{display:grid;gap:4px;}small{color:${props => props.theme.colors.subtle};}`
const SecondaryButton = styled(Button)`color:${props => props.theme.colors.text};background:transparent;border:1px solid ${props => props.theme.colors.borderStrong};`
const HelpText = styled.p`color:${props => props.theme.colors.muted};font-size:14px;line-height:1.5;`
const ButtonRow = styled.div`display:flex;align-items:center;flex-wrap:wrap;gap:12px;margin-top:18px;`

export default function Account() {
  const { user, logout } = useAuth()
  const { themeName, setThemeName } = useAppTheme()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [profile, setProfile] = useState<PlaybackProfile>({
    replaygain_mode: 'track', replaygain_preamp_db: 0, transcode_enabled: false,
    transcode_format: 'mp3', transcode_bitrate: 192,
  })
  const [sessions, setSessions] = useState<UserSession[]>([])
  const [currentSessionID, setCurrentSessionID] = useState('')
  const [profileSaving, setProfileSaving] = useState(false)
  const [scrobbleSaving, setScrobbleSaving] = useState(false)
  const [scrobbleLoaded, setScrobbleLoaded] = useState(false)
  const [scrobbleStatus, setScrobbleStatus] = useState('')
  const scrobbleSaveTimer = useRef<number | null>(null)
  const [scrobbleSettings, setScrobbleSettings] = useState<ScrobbleSettings>({
    listenbrainz_enabled: false,
    has_listenbrainz_token: false,
    listenbrainz_token: '',
    lastfm_enabled: false,
    lastfm_server_configured: false,
    lastfm_username: '',
    has_lastfm_session_key: false,
    has_lastfm_pending_token: false,
  })

  const loadAccountSettings = async () => {
    const [loadedProfile, sessionData, loadedScrobbleSettings] = await Promise.all([
      accountAPI.getPlaybackProfile(), accountAPI.getSessions(), accountAPI.getScrobbleSettings(),
    ])
    setProfile(loadedProfile)
    setSessions(sessionData.sessions)
    setCurrentSessionID(sessionData.current_session_id)
    setScrobbleSettings({
      ...loadedScrobbleSettings,
      listenbrainz_token: '',
    })
    setScrobbleLoaded(true)
  }

  useEffect(() => { void loadAccountSettings() }, [])

  const saveProfile = async () => {
    setProfileSaving(true)
    setError('')
    try {
      setProfile(await accountAPI.savePlaybackProfile(profile))
      setMessage('Playback settings saved. They apply to the next track you play.')
    } catch {
      setError('Playback settings could not be saved.')
    } finally {
      setProfileSaving(false)
    }
  }

  const listenBrainzEnabled = scrobbleSettings.listenbrainz_enabled
  const listenBrainzToken = scrobbleSettings.listenbrainz_token
  const hasListenBrainzToken = scrobbleSettings.has_listenbrainz_token
  const lastFMEnabled = scrobbleSettings.lastfm_enabled
  const lastFMServerConfigured = scrobbleSettings.lastfm_server_configured
  const lastFMUsername = scrobbleSettings.lastfm_username
  const hasLastFMSessionKey = scrobbleSettings.has_lastfm_session_key
  const hasLastFMPendingToken = scrobbleSettings.has_lastfm_pending_token

  useEffect(() => {
    if (!scrobbleLoaded) return
    if (scrobbleSaveTimer.current) {
      window.clearTimeout(scrobbleSaveTimer.current)
    }

    scrobbleSaveTimer.current = window.setTimeout(() => {
      const settingsToSave: ScrobbleSettings = {
        listenbrainz_enabled: listenBrainzEnabled,
        has_listenbrainz_token: hasListenBrainzToken,
        listenbrainz_token: listenBrainzToken,
        lastfm_enabled: lastFMEnabled,
        lastfm_server_configured: lastFMServerConfigured,
        lastfm_username: lastFMUsername,
        has_lastfm_session_key: hasLastFMSessionKey,
        has_lastfm_pending_token: hasLastFMPendingToken,
      }

      setScrobbleSaving(true)
      setError('')
      void accountAPI.saveScrobbleSettings(settingsToSave)
        .then(saved => {
          setScrobbleSettings(current => ({
            ...current,
            ...saved,
            listenbrainz_token: '',
          }))
          setScrobbleStatus('Saved')
        })
        .catch((requestError: unknown) => {
          const responseError = requestError as { response?: { data?: { error?: string } } }
          setError(responseError.response?.data?.error || 'Scrobbling settings could not be saved.')
          setScrobbleStatus('')
        })
        .finally(() => setScrobbleSaving(false))
    }, 650)

    return () => {
      if (scrobbleSaveTimer.current) {
        window.clearTimeout(scrobbleSaveTimer.current)
      }
    }
  }, [
    scrobbleLoaded,
    listenBrainzEnabled,
    listenBrainzToken,
    hasListenBrainzToken,
    lastFMEnabled,
    lastFMServerConfigured,
    lastFMUsername,
    hasLastFMSessionKey,
    hasLastFMPendingToken,
  ])

  const startLastFMConnection = async () => {
    setScrobbleSaving(true)
    setError('')
    setMessage('')
    try {
      const result = await accountAPI.startLastFMAuth()
      const refreshed = await accountAPI.getScrobbleSettings()
      setScrobbleSettings(current => ({
        ...current,
        ...refreshed,
      }))
      window.open(result.auth_url, '_blank', 'noopener,noreferrer')
      setScrobbleStatus('Approve Last.fm in the new tab, then click Complete connection.')
    } catch (requestError: unknown) {
      const responseError = requestError as { response?: { data?: { error?: string } }; message?: string }
      setError(responseError.response?.data?.error || responseError.message || 'Last.fm connection could not be started.')
    } finally {
      setScrobbleSaving(false)
    }
  }

  const finishLastFMConnection = async () => {
    setScrobbleSaving(true)
    setError('')
    setMessage('')
    try {
      const saved = await accountAPI.completeLastFMAuth()
      setScrobbleSettings({
        ...saved,
        listenbrainz_token: '',
      })
      setScrobbleStatus(saved.lastfm_username ? `Connected as ${saved.lastfm_username}` : 'Connected')
    } catch (requestError: unknown) {
      const responseError = requestError as { response?: { data?: { error?: string } }; message?: string }
      setError(responseError.response?.data?.error || responseError.message || 'Last.fm connection could not be completed.')
    } finally {
      setScrobbleSaving(false)
    }
  }

  const disconnectLastFMConnection = async () => {
    if (!window.confirm('Disconnect Last.fm from this account?')) return
    setScrobbleSaving(true)
    setError('')
    setMessage('')
    try {
      const saved = await accountAPI.disconnectLastFM()
      setScrobbleSettings({
        ...saved,
        listenbrainz_token: '',
      })
      setScrobbleStatus('Disconnected')
    } catch (requestError: unknown) {
      const responseError = requestError as { response?: { data?: { error?: string } }; message?: string }
      setError(responseError.response?.data?.error || responseError.message || 'Last.fm could not be disconnected.')
    } finally {
      setScrobbleSaving(false)
    }
  }

  const revokeSession = async (session: UserSession) => {
    await accountAPI.revokeSession(session.id)
    if (session.id === currentSessionID) {
      logout()
      return
    }
    await loadAccountSettings()
  }

  const changePassword = async (event: React.FormEvent) => {
    event.preventDefault()
    setMessage('')
    setError('')
    if (newPassword.length < 8) {
      setError('Use at least 8 characters for the new password.')
      return
    }
    if (newPassword !== confirmPassword) {
      setError('The new passwords do not match.')
      return
    }
    try {
      setSaving(true)
      await api.put('/auth/password', {
        current_password: currentPassword,
        new_password: newPassword,
      })
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setMessage('Password changed successfully.')
    } catch (requestError: unknown) {
      const responseError = requestError as { response?: { data?: { error?: string } } }
      setError(responseError.response?.data?.error || 'Could not change your password.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Container>
      <h1>Account</h1>
      <Card>
        <h2><UserRound size={20} /> Profile</h2>
        <p>{user?.username}</p>
        <p>{user?.email}</p>
        <p>Role: {user?.role}</p>
      </Card>

      <Card>
        <h2><Palette size={20} /> Appearance</h2>
        <p>Choose how WaveNode looks on this device.</p>
        <SettingsGrid>
          <Setting>Theme
            <select value={themeName} onChange={event => setThemeName(event.target.value as AppThemeName)}>
              {Object.values(appThemes).map(theme => (
                <option key={theme.name} value={theme.name}>{theme.label}</option>
              ))}
            </select>
          </Setting>
          <Setting>Style
            <input readOnly value={appThemes[themeName].description} />
          </Setting>
        </SettingsGrid>
      </Card>

      <Card>
        <h2><KeyRound size={20} /> Change password</h2>
        <Form onSubmit={changePassword}>
          <label>
            Current password
            <input type="password" autoComplete="current-password" value={currentPassword} onChange={event => setCurrentPassword(event.target.value)} required />
          </label>
          <label>
            New password
            <input type="password" autoComplete="new-password" value={newPassword} onChange={event => setNewPassword(event.target.value)} minLength={8} required />
          </label>
          <label>
            Confirm new password
            <input type="password" autoComplete="new-password" value={confirmPassword} onChange={event => setConfirmPassword(event.target.value)} minLength={8} required />
          </label>
          <Button type="submit" disabled={saving}>{saving ? 'Saving...' : 'Change password'}</Button>
        </Form>
        {message && <Message>{message}</Message>}
        {error && <Message $error>{error}</Message>}
      </Card>

      <Card>
        <h2><Headphones size={20} /> Playback quality</h2>
        <p>Normalize volume and reduce bandwidth for mobile or remote listening.</p>
        <SettingsGrid>
          <Setting>ReplayGain
            <select value={profile.replaygain_mode} onChange={event => setProfile(current => ({ ...current, replaygain_mode: event.target.value as PlaybackProfile['replaygain_mode'] }))}>
              <option value="off">Off</option><option value="track">Track mode</option><option value="album">Album mode</option>
            </select>
          </Setting>
          <Setting>Preamp adjustment (dB)
            <input type="number" min={-15} max={15} step={0.5} value={profile.replaygain_preamp_db} onChange={event => setProfile(current => ({ ...current, replaygain_preamp_db: Number(event.target.value) }))} />
          </Setting>
          <Setting>Remote format
            <select value={profile.transcode_format} onChange={event => setProfile(current => ({ ...current, transcode_format: event.target.value as PlaybackProfile['transcode_format'] }))}>
              <option value="mp3">MP3</option><option value="opus">Opus</option><option value="aac">AAC</option>
            </select>
          </Setting>
          <Setting>Bitrate
            <select value={profile.transcode_bitrate} onChange={event => setProfile(current => ({ ...current, transcode_bitrate: Number(event.target.value) }))}>
              {[64, 96, 128, 160, 192, 256, 320].map(value => <option key={value} value={value}>{value} kbps</option>)}
            </select>
          </Setting>
        </SettingsGrid>
        <CheckSetting><input type="checkbox" checked={profile.transcode_enabled} onChange={event => setProfile(current => ({ ...current, transcode_enabled: event.target.checked }))} />Always transcode web playback using this profile</CheckSetting>
        <Button type="button" onClick={() => void saveProfile()} disabled={profileSaving}>{profileSaving ? <RefreshCw size={16} /> : <Headphones size={16} />} Save playback settings</Button>
      </Card>

      <Card>
        <h2><Radio size={20} /> Scrobbling</h2>
        <HelpText>Send listening activity to ListenBrainz and Last.fm. Changes save automatically.</HelpText>
        <CheckSetting>
          <input
            type="checkbox"
            checked={scrobbleSettings.listenbrainz_enabled}
            onChange={event => setScrobbleSettings(current => ({ ...current, listenbrainz_enabled: event.target.checked }))}
          />
          Enable ListenBrainz scrobbling
        </CheckSetting>
        <Setting>ListenBrainz user token
          <input
            type="password"
            placeholder={scrobbleSettings.has_listenbrainz_token ? 'Saved. Leave blank to keep existing token.' : 'Paste ListenBrainz token'}
            value={scrobbleSettings.listenbrainz_token || ''}
            onChange={event => setScrobbleSettings(current => ({ ...current, listenbrainz_token: event.target.value }))}
          />
        </Setting>
        {!scrobbleSettings.lastfm_server_configured && (
          <HelpText>Last.fm is not configured by an administrator yet.</HelpText>
        )}
        {scrobbleSettings.has_lastfm_session_key && (
          <HelpText>Last.fm is connected{scrobbleSettings.lastfm_username ? ` as ${scrobbleSettings.lastfm_username}` : ''}.</HelpText>
        )}
        {scrobbleSettings.has_lastfm_pending_token && (
          <HelpText>Last.fm is waiting for approval. Approve WaveNode in the Last.fm tab, then complete the connection.</HelpText>
        )}
        <ButtonRow>
          {scrobbleSettings.has_lastfm_session_key ? (
            <Button $danger type="button" onClick={() => void disconnectLastFMConnection()} disabled={scrobbleSaving}>Disconnect Last.fm</Button>
          ) : scrobbleSettings.has_lastfm_pending_token ? (
            <Button type="button" onClick={() => void finishLastFMConnection()} disabled={scrobbleSaving}>Complete Last.fm connection</Button>
          ) : (
            <Button type="button" onClick={() => void startLastFMConnection()} disabled={scrobbleSaving || !scrobbleSettings.lastfm_server_configured}>Connect Last.fm</Button>
          )}
          <HelpText>{scrobbleSaving ? 'Saving...' : scrobbleStatus}</HelpText>
        </ButtonRow>
      </Card>

      <Card>
        <h2><MonitorSmartphone size={20} /> Connected devices</h2>
        <p>Revoke devices that should no longer have access to your account.</p>
        <SessionList>
          {sessions.map(session => (
            <SessionRow key={session.id}>
              <div><strong>{session.device_name}{session.id === currentSessionID ? ' · This device' : ''}</strong><small>{session.ip_address} · Active {new Date(session.last_seen_at).toLocaleString()}</small></div>
              <SecondaryButton type="button" onClick={() => void revokeSession(session)}><XCircle size={15} /> Revoke</SecondaryButton>
            </SessionRow>
          ))}
        </SessionList>
        {sessions.length > 1 && <SecondaryButton type="button" onClick={async () => { await accountAPI.revokeOtherSessions(); await loadAccountSettings() }}><LogOut size={15} /> Sign out other devices</SecondaryButton>}
      </Card>

      <Card>
        <h2>Session</h2>
        <p>Sign out on this device and return to the login screen.</p>
        <Button $danger type="button" onClick={logout}><LogOut size={16} /> Sign out</Button>
      </Card>
    </Container>
  )
}
