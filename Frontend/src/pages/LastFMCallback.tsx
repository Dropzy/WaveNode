import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import styled from 'styled-components'
import { CheckCircle2, Loader2, XCircle } from 'lucide-react'
import { accountAPI } from '../services/api'

export default function LastFMCallback() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>('loading')
  const [message, setMessage] = useState('Completing Last.fm connection...')

  useEffect(() => {
    const token = searchParams.get('token') || ''
    if (!token) {
      setStatus('error')
      setMessage('Last.fm did not return an auth token. Start the connection again from Account.')
      return
    }

    let active = true
    void accountAPI.completeLastFMAuth(token)
      .then(settings => {
        if (!active) return
        setStatus('success')
        setMessage(settings.lastfm_username ? `Last.fm connected as ${settings.lastfm_username}.` : 'Last.fm connected.')
        window.setTimeout(() => navigate('/account', { replace: true }), 1400)
      })
      .catch((requestError: unknown) => {
        if (!active) return
        const responseError = requestError as { response?: { data?: { error?: string } }; message?: string }
        setStatus('error')
        setMessage(responseError.response?.data?.error || responseError.message || 'Last.fm connection could not be completed.')
      })

    return () => {
      active = false
    }
  }, [navigate, searchParams])

  const Icon = status === 'loading' ? Loader2 : status === 'success' ? CheckCircle2 : XCircle

  return (
    <CallbackState $status={status}>
      <Icon className={status === 'loading' ? 'spin' : undefined} size={34} />
      <h1>Last.fm Connection</h1>
      <p>{message}</p>
      {status === 'error' && (
        <button type="button" onClick={() => navigate('/account', { replace: true })}>
          Back to Account
        </button>
      )}
    </CallbackState>
  )
}

const CallbackState = styled.main<{ $status: 'loading' | 'success' | 'error' }>`
  min-height: 100%;
  display: grid;
  place-content: center;
  justify-items: center;
  gap: 14px;
  padding: 32px;
  text-align: center;

  svg {
    color: ${({ theme, $status }) => $status === 'error' ? theme.colors.danger : theme.colors.accent};
  }

  h1 {
    margin: 0;
    color: ${({ theme }) => theme.colors.text};
  }

  p {
    max-width: 440px;
    margin: 0;
    color: ${({ theme }) => theme.colors.muted};
    line-height: 1.5;
  }

  button {
    margin-top: 8px;
    padding: 10px 16px;
    border-radius: 999px;
    color: ${({ theme }) => theme.colors.accentText};
    background: ${({ theme }) => theme.colors.accentGradient};
    font-weight: 800;
  }
`
