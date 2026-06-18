import { useEffect, useState } from 'react'
import styled from 'styled-components'
import { Clock3, Download, Play, Search, Trash2 } from 'lucide-react'
import { historyAPI, type ListeningHistoryEntry } from '../services/api'
import { useAudio } from '../contexts/AudioContext'
import { getTrackArtworkUrl } from '../utils/mediaUrl'

export default function History() {
  const { playFromQueue } = useAudio()
  const [entries, setEntries] = useState<ListeningHistoryEntry[]>([])
  const [search, setSearch] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setLoading(true)
      setError('')
      void historyAPI.get(search)
        .then(setEntries)
        .catch(() => setError('Listening history could not be loaded.'))
        .finally(() => setLoading(false))
    }, 250)
    return () => window.clearTimeout(timer)
  }, [search])

  const clear = async () => {
    if (!window.confirm('Clear your entire listening history? This cannot be undone.')) return
    await historyAPI.clear()
    setEntries([])
  }

  return (
    <Page>
      <Header>
        <div><h1><Clock3 /> Listening history</h1><p>Every completed play from WaveNode and connected Subsonic clients.</p></div>
        <Actions>
          <Button onClick={() => void historyAPI.exportCSV()}><Download size={16} /> Export CSV</Button>
          <DangerButton onClick={() => void clear()}><Trash2 size={16} /> Clear history</DangerButton>
        </Actions>
      </Header>
      <SearchBox><Search size={18} /><input value={search} onChange={event => setSearch(event.target.value)} placeholder="Search title, artist, or album" /></SearchBox>
      {error && <ErrorText>{error}</ErrorText>}
      {loading ? <Empty>Loading history...</Empty> : entries.length === 0 ? <Empty>No matching plays.</Empty> : (
        <List>
          {entries.map((entry, index) => (
            <Row key={entry.id} onDoubleClick={() => playFromQueue(entries.map(item => item.track), index)}>
              <Artwork $url={getTrackArtworkUrl(entry.track)}><Play size={16} /></Artwork>
              <Track><strong>{entry.track.title}</strong><span>{entry.track.artist} · {entry.track.album}</span></Track>
              <Source>{entry.source}{entry.device ? ` · ${entry.device}` : ''}</Source>
              <Time>{new Date(entry.played_at).toLocaleString()}</Time>
            </Row>
          ))}
        </List>
      )}
    </Page>
  )
}

const Page = styled.main`padding:28px clamp(18px,3vw,44px) 70px;overflow-y:auto;color:#fff;`
const Header = styled.header`display:flex;justify-content:space-between;align-items:flex-start;gap:20px;margin-bottom:22px;h1{display:flex;align-items:center;gap:10px;font-size:clamp(28px,4vw,42px);}p{color:#aeb5b0;margin-top:6px;}@media(max-width:760px){flex-direction:column;}`
const Actions = styled.div`display:flex;flex-wrap:wrap;gap:9px;`
const Button = styled.button`display:flex;align-items:center;gap:7px;padding:10px 14px;border:1px solid #505752;border-radius:999px;color:#fff;&:hover{border-color:#1ed760;}`
const DangerButton = styled(Button)`border-color:#71383d;color:#ff9ca2;`
const SearchBox = styled.label`display:flex;align-items:center;gap:10px;max-width:560px;padding:10px 14px;border-radius:999px;background:#242725;color:#aaa;margin-bottom:18px;input{flex:1;background:none;border:0;color:#fff;outline:none;}`
const List = styled.div`display:grid;gap:7px;`
const Row = styled.div`display:grid;grid-template-columns:48px minmax(180px,1fr) minmax(130px,.5fr) minmax(170px,.5fr);align-items:center;gap:14px;padding:10px 14px;border-radius:9px;background:#181a19;&:hover{background:#242725;}@media(max-width:760px){grid-template-columns:44px 1fr;.history-source{display:none;}}`
const Artwork = styled.div<{ $url?: string }>`width:44px;height:44px;display:grid;place-items:center;border-radius:5px;background:${props => props.$url ? `url("${props.$url}") center/cover` : '#315c40'};color:transparent;&:hover{color:#fff;background-blend-mode:multiply;}`
const Track = styled.div`display:grid;gap:4px;min-width:0;strong,span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;}span{font-size:12px;color:#aeb5b0;}`
const Source = styled.div.attrs({ className: 'history-source' })`font-size:12px;color:#9ca39e;text-transform:capitalize;`
const Time = styled.time`font-size:12px;color:#b8beb9;text-align:right;`
const Empty = styled.div`padding:70px;text-align:center;color:#929a94;`
const ErrorText = styled.p`color:#ff858d;`
