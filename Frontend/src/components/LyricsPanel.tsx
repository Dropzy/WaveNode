import { useEffect, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { Mic2, X } from 'lucide-react'
import styled from 'styled-components'
import { musicAPI, type Lyrics, type Music } from '../services/api'

interface LyricsPanelProps {
  isOpen: boolean
  track: Music | null
  currentTime: number
  onSeek: (seconds: number) => void
  onClose: () => void
}

const Panel = styled.aside<{ $open: boolean }>`
  position: fixed;
  inset: 0 0 104px auto;
  width: min(440px, 100vw);
  z-index: 1100;
  display: flex;
  flex-direction: column;
  background: ${({ theme }) => theme.colors.backgroundElevated};
  border-left: 1px solid ${({ theme }) => theme.colors.border};
  box-shadow: -18px 0 55px ${({ theme }) => theme.colors.shadow};
  transform: translateX(${({ $open }) => $open ? '0' : '100%'});
  transition: transform 0.25s ease;
  pointer-events: ${({ $open }) => $open ? 'auto' : 'none'};

  @media (max-width: 768px) {
    bottom: 80px;
    width: 100%;
  }
`

const Header = styled.header`
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 18px;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
  background: ${({ theme }) => theme.colors.surface};
`

const HeaderText = styled.div`
  flex: 1;
  min-width: 0;

  h2, p { margin: 0; }
  h2 { color: ${({ theme }) => theme.colors.text}; font-size: 18px; }
  p {
    margin-top: 3px;
    color: ${({ theme }) => theme.colors.muted};
    font-size: 12px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
`

const CloseButton = styled.button`
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 0;
  background: transparent;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;

  &:hover { color: ${({ theme }) => theme.colors.text}; }
`

const Body = styled.div`
  flex: 1;
  overflow-y: auto;
  padding: 28px 24px 56px;
  scrollbar-color: ${({ theme }) => theme.colors.borderStrong} transparent;
`

const Message = styled.div`
  min-height: 100%;
  display: grid;
  place-items: center;
  text-align: center;
  color: ${({ theme }) => theme.colors.muted};
  font-size: 14px;
`

const SyncedLine = styled.button<{ $active: boolean; $past: boolean }>`
  display: block;
  width: 100%;
  border: 0;
  padding: 9px 0;
  background: transparent;
  color: ${({ theme, $active, $past }) => $active ? theme.colors.accent : $past ? theme.colors.text : theme.colors.muted};
  font: inherit;
  font-size: ${({ $active }) => $active ? '21px' : '17px'};
  font-weight: ${({ $active }) => $active ? 800 : 600};
  line-height: 1.35;
  text-align: left;
  cursor: pointer;
  transition: color 0.18s ease, font-size 0.18s ease;

  &:hover { color: ${({ theme }) => theme.colors.text}; }
`

const PlainLyrics = styled.div`
  color: ${({ theme }) => theme.colors.text};
  white-space: pre-wrap;
  font-size: 16px;
  line-height: 1.75;
`

export function LyricsPanel({ isOpen, track, currentTime, onSeek, onClose }: LyricsPanelProps) {
  const [lyrics, setLyrics] = useState<Lyrics | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const lineRefs = useRef<Record<number, HTMLButtonElement | null>>({})

  useEffect(() => {
    if (!isOpen || !track || track.is_external) return
    let active = true
    setLoading(true)
    setError('')
    setLyrics(null)
    void musicAPI.getLyrics(track.id)
      .then(result => { if (active) setLyrics(result) })
      .catch(cause => { if (active) setError(cause instanceof Error ? cause.message : 'Lyrics could not be loaded') })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [isOpen, track])

  const activeIndex = useMemo(() => {
    if (!lyrics?.synced) return -1
    const positionMS = currentTime * 1000
    let index = -1
    for (let i = 0; i < lyrics.lines.length; i += 1) {
      if (lyrics.lines[i].time_ms > positionMS) break
      index = i
    }
    return index
  }, [currentTime, lyrics])

  useEffect(() => {
    if (activeIndex >= 0) lineRefs.current[activeIndex]?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [activeIndex])

  const content = (() => {
    if (loading) return <Message>Loading lyrics...</Message>
    if (error) return <Message>{error}</Message>
    if (lyrics?.instrumental) return <Message>Instrumental track</Message>
    if (!lyrics?.available) return <Message>Lyrics are not available for this track.</Message>
    if (lyrics.synced && lyrics.lines.length > 0) {
      return lyrics.lines.map((line, index) => (
        <SyncedLine
          key={`${line.time_ms}:${index}`}
          ref={element => { lineRefs.current[index] = element }}
          $active={index === activeIndex}
          $past={index < activeIndex}
          onClick={() => onSeek(line.time_ms / 1000)}
        >
          {line.text || '\u00a0'}
        </SyncedLine>
      ))
    }
    return <PlainLyrics>{lyrics.plain_text}</PlainLyrics>
  })()

  return createPortal(
    <Panel $open={isOpen} aria-hidden={!isOpen}>
      <Header>
        <Mic2 size={20} />
        <HeaderText><h2>Lyrics</h2><p>{track ? `${track.title} - ${track.artist}` : ''}</p></HeaderText>
        <CloseButton onClick={onClose} aria-label="Close lyrics"><X /></CloseButton>
      </Header>
      <Body>{content}</Body>
    </Panel>,
    document.body,
  )
}
