import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { Headphones, KeyRound, LogOut, MonitorSmartphone, RefreshCw, UserRound, XCircle } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { accountAPI, api, type PlaybackProfile, type UserSession } from '../services/api'

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
  border: 1px solid #333;
  border-radius: 12px;
  background: #181818;
`

const Form = styled.form`
  display: grid;
  gap: 14px;
  margin-top: 18px;

  label {
    display: grid;
    gap: 7px;
    color: #ddd;
    font-size: 14px;
  }

  input {
    padding: 11px 12px;
    border: 1px solid #454545;
    border-radius: 8px;
    color: #fff;
    background: #242424;
  }
`

const Button = styled.button<{ $danger?: boolean }>`
  width: fit-content;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-radius: 999px;
  color: ${props => props.$danger ? '#fff' : '#07130b'};
  background: ${props => props.$danger ? '#b3261e' : '#1ed760'};
  font-weight: 800;
`

const Message = styled.p<{ $error?: boolean }>`
  margin-top: 12px;
  color: ${props => props.$error ? '#ff7b7b' : '#65e98f'};
`

const SettingsGrid = styled.div`display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px;margin-top:18px;@media(max-width:640px){grid-template-columns:1fr;}`
const Setting = styled.label`display:grid;gap:7px;color:#ddd;font-size:14px;select,input{padding:11px 12px;border:1px solid #454545;border-radius:8px;color:#fff;background:#242424;}`
const CheckSetting = styled.label`display:flex;align-items:center;gap:10px;margin:16px 0;color:#ddd;input{accent-color:#1ed760;}`
const SessionList = styled.div`display:grid;gap:9px;margin:16px 0;`
const SessionRow = styled.div`display:flex;justify-content:space-between;align-items:center;gap:15px;padding:13px;border:1px solid #333;border-radius:9px;background:#202020;div{display:grid;gap:4px;}small{color:#999;}`
const SecondaryButton = styled(Button)`color:#fff;background:transparent;border:1px solid #555;`

export default function Account() {
  const { user, logout } = useAuth()
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

  const loadAccountSettings = async () => {
    const [loadedProfile, sessionData] = await Promise.all([
      accountAPI.getPlaybackProfile(), accountAPI.getSessions(),
    ])
    setProfile(loadedProfile)
    setSessions(sessionData.sessions)
    setCurrentSessionID(sessionData.current_session_id)
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
