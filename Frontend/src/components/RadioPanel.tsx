import { useCallback, useEffect, useMemo, useState } from 'react'
import styled from 'styled-components'
import { Heart, MapPin, Play, Radio, Search } from 'lucide-react'
import { radioAPI, type Music, type RadioHomeResponse, type RadioStation } from '../services/api'
import { useAudio } from '../contexts/AudioContext'
import { getArtworkGradient } from '../utils/mediaUrl'

interface RadioPanelProps {
  query: string
}

const genres = ['Electronic', 'Rock', 'Jazz', 'Classical', 'News', 'Talk', 'Hip Hop', 'Country', 'Ambient']

export const radioStationToTrack = (station: RadioStation): Music => ({
  id: `radio:${station.id}`,
  title: station.name,
  artist: station.country || station.language || 'Internet radio',
  album: station.tags?.split(',')[0] || 'Live radio',
  genre: 'Radio',
  duration: 0,
  release_date: '',
  file_path: '',
  image_url: station.favicon_url,
  created_at: '',
  updated_at: '',
  stream_url: station.stream_url,
  is_external: true,
  external_kind: 'radio',
  radio_station_id: station.id,
})

const RadioPanel = ({ query }: RadioPanelProps) => {
  const { playFromQueue } = useAudio()
  const [home, setHome] = useState<RadioHomeResponse>({ favourites: [], popular: [], trending: [], local: [] })
  const [results, setResults] = useState<RadioStation[]>([])
  const [selectedGenre, setSelectedGenre] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [favoriteAction, setFavoriteAction] = useState<string | null>(null)

  const country = useMemo(() => {
    const parts = navigator.language.split('-')
    return parts.length > 1 ? parts[1].toUpperCase() : ''
  }, [])

  const loadHome = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await radioAPI.getHome(country)
      setHome(response)
      if (response.directory_error && response.popular.length === 0 && response.trending.length === 0) {
        setError('The radio directory is temporarily unavailable. Your favourites are still available.')
      }
    } catch (requestError) {
      console.error('Failed to load radio stations:', requestError)
      setError('Radio stations could not be loaded.')
    } finally {
      setLoading(false)
    }
  }, [country])

  useEffect(() => {
    if (query.trim() || selectedGenre) return
    void loadHome()
  }, [loadHome, query, selectedGenre])

  useEffect(() => {
    const search = query.trim()
    if (!search && !selectedGenre) {
      setResults([])
      return
    }
    let active = true
    const timeout = window.setTimeout(async () => {
      setLoading(true)
      setError(null)
      try {
        const stations = await radioAPI.search({
          q: search || undefined,
          tag: selectedGenre || undefined,
          limit: 80,
          order: search ? 'clickcount' : 'votes',
        })
        if (active) setResults(stations)
      } catch (requestError) {
        console.error('Failed to search radio stations:', requestError)
        if (active) setError('Radio search could not be loaded.')
      } finally {
        if (active) setLoading(false)
      }
    }, 350)
    return () => {
      active = false
      window.clearTimeout(timeout)
    }
  }, [query, selectedGenre])

  const updateStationEverywhere = (stationId: string, favourite: boolean, saved?: RadioStation) => {
    const update = (stations: RadioStation[]) => stations.map(station => station.id === stationId
      ? { ...station, ...saved, favourite }
      : station)
    setResults(update)
    setHome(previous => {
      const all = [previous.favourites, previous.popular, previous.trending, previous.local]
        .flat()
      const station = saved || all.find(item => item.id === stationId)
      return {
        ...previous,
        favourites: favourite && station
          ? [{ ...station, favourite: true }, ...previous.favourites.filter(item => item.id !== stationId)]
          : previous.favourites.filter(item => item.id !== stationId),
        popular: update(previous.popular),
        trending: update(previous.trending),
        local: update(previous.local),
      }
    })
    window.dispatchEvent(new CustomEvent('wavenode:radio-favorites-changed'))
  }

  const toggleFavorite = async (station: RadioStation) => {
    try {
      setFavoriteAction(station.id)
      if (station.favourite) {
        await radioAPI.removeFavorite(station.id)
        updateStationEverywhere(station.id, false)
      } else {
        const saved = await radioAPI.addFavorite(station.id)
        updateStationEverywhere(station.id, true, saved)
      }
    } catch (requestError) {
      console.error('Failed to update radio favourite:', requestError)
      setError('The radio favourite could not be updated.')
    } finally {
      setFavoriteAction(null)
    }
  }

  const playStation = (station: RadioStation, stations: RadioStation[]) => {
    void radioAPI.registerClick(station.id).catch(() => undefined)
    const queue = stations.map(radioStationToTrack)
    playFromQueue(queue, Math.max(0, stations.findIndex(item => item.id === station.id)))
  }

  const renderStations = (stations: RadioStation[], horizontal = false) => {
    const Container = horizontal ? StationRow : StationGrid
    return (
      <Container>
        {stations.map(station => (
          <StationCard
            key={station.id}
            role="button"
            tabIndex={0}
            onClick={() => playStation(station, stations)}
            onKeyDown={event => {
              if (event.key === 'Enter' || event.key === ' ') playStation(station, stations)
            }}
          >
            <StationArtwork $imageUrl={station.favicon_url} $fallback={getArtworkGradient(station.id)}>
              {!station.favicon_url && <Radio size={34} />}
              <PlayBadge><Play size={17} fill="currentColor" /></PlayBadge>
            </StationArtwork>
            <StationInfo>
              <strong>{station.name}</strong>
              <span><MapPin size={12} /> {station.country || station.language || 'Internet radio'}</span>
              <small>{[station.codec, station.bitrate ? `${station.bitrate} kbps` : ''].filter(Boolean).join(' · ') || 'Live stream'}</small>
            </StationInfo>
            <FavoriteButton
              type="button"
              $active={station.favourite}
              disabled={favoriteAction === station.id}
              aria-label={station.favourite ? `Remove ${station.name} from favourites` : `Add ${station.name} to favourites`}
              onClick={event => {
                event.stopPropagation()
                void toggleFavorite(station)
              }}
            >
              <Heart size={18} fill={station.favourite ? 'currentColor' : 'none'} />
            </FavoriteButton>
          </StationCard>
        ))}
      </Container>
    )
  }

  const searching = Boolean(query.trim() || selectedGenre)
  return (
    <Panel>
      <PanelHeader>
        <div><h2>Internet radio</h2><p>Live stations from around the world.</p></div>
      </PanelHeader>
      <GenreBar>
        <GenreButton type="button" $active={!selectedGenre} onClick={() => setSelectedGenre('')}>All</GenreButton>
        {genres.map(genre => (
          <GenreButton key={genre} type="button" $active={selectedGenre === genre} onClick={() => setSelectedGenre(selectedGenre === genre ? '' : genre)}>{genre}</GenreButton>
        ))}
      </GenreBar>
      {error && <Message $error>{error}</Message>}
      {loading ? <Message>Loading radio stations...</Message> : searching ? (
        results.length ? renderStations(results) : <Empty><Search size={38} /><strong>No stations found</strong><span>Try another station name or genre.</span></Empty>
      ) : (
        <>
          {home.favourites.length > 0 && <StationSection><h3>Favourite stations</h3>{renderStations(home.favourites, true)}</StationSection>}
          {home.local.length > 0 && <StationSection><h3>Popular near you</h3>{renderStations(home.local, true)}</StationSection>}
          {home.trending.length > 0 && <StationSection><h3>Trending now</h3>{renderStations(home.trending, true)}</StationSection>}
          {home.popular.length > 0 && <StationSection><h3>Popular worldwide</h3>{renderStations(home.popular)}</StationSection>}
          {!home.favourites.length && !home.local.length && !home.trending.length && !home.popular.length && !error && (
            <Empty><Radio size={42} /><strong>No stations available</strong><span>Try again in a moment.</span></Empty>
          )}
        </>
      )}
    </Panel>
  )
}

const Panel = styled.div`padding: 4px 0 28px;`
const PanelHeader = styled.div`display:flex;justify-content:space-between;gap:16px;margin-bottom:18px;h2{font-size:24px;margin:0 0 5px;}p{margin:0;color:${({ theme }) => theme.colors.muted};}`
const GenreBar = styled.div`display:flex;gap:8px;overflow-x:auto;padding:2px 0 14px;margin-bottom:8px;`
const GenreButton = styled.button<{ $active: boolean }>`flex:0 0 auto;padding:8px 13px;border:1px solid ${({ theme, $active }) => $active ? theme.colors.accent : theme.colors.border};border-radius:999px;background:${({ theme, $active }) => $active ? theme.colors.accentSoft : theme.colors.backgroundElevated};color:${({ theme }) => theme.colors.text};font-weight:700;`
const StationSection = styled.section`margin:24px 0 34px;h3{margin:0 0 14px;font-size:19px;}`
const StationGrid = styled.div`display:grid;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:13px;`
const StationRow = styled.div`display:flex;gap:13px;overflow-x:auto;padding:2px 2px 12px;`
const StationCard = styled.div`position:relative;min-width:0;display:grid;grid-template-columns:72px minmax(0,1fr) auto;align-items:center;gap:12px;padding:12px;border:1px solid ${({ theme }) => theme.colors.border};border-radius:12px;background:${({ theme }) => theme.colors.backgroundElevated};cursor:pointer;transition:.18s ease;&:hover{background:${({ theme }) => theme.colors.controlBg};transform:translateY(-2px);}${StationRow} &{flex:0 0 min(330px,82vw);}`
const StationArtwork = styled.div<{ $imageUrl?: string; $fallback: string }>`position:relative;width:72px;height:72px;display:grid;place-items:center;overflow:hidden;border-radius:9px;background:${props => props.$imageUrl ? `url("${props.$imageUrl}") center/cover no-repeat` : props.$fallback};`
const PlayBadge = styled.div`position:absolute;inset:auto 6px 6px auto;width:30px;height:30px;display:grid;place-items:center;border-radius:50%;background:${({ theme }) => theme.colors.accent};color:${({ theme }) => theme.colors.accentText};opacity:0;transition:.18s;${StationCard}:hover &{opacity:1;}`
const StationInfo = styled.div`min-width:0;strong,span,small{display:flex;align-items:center;gap:4px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;}strong{margin-bottom:6px;}span{color:${({ theme }) => theme.colors.muted};font-size:13px;}small{margin-top:5px;color:${({ theme }) => theme.colors.subtle};font-size:11px;}`
const FavoriteButton = styled.button<{ $active: boolean }>`align-self:start;padding:8px;border-radius:50%;color:${({ theme, $active }) => $active ? theme.colors.danger : theme.colors.muted};background:rgba(0,0,0,.22);&:hover{color:${({ theme }) => theme.colors.danger};}&:disabled{opacity:.45;}`
const Message = styled.div<{ $error?: boolean }>`padding:28px;color:${({ theme, $error }) => $error ? theme.colors.danger : theme.colors.muted};text-align:center;`
const Empty = styled.div`min-height:240px;display:grid;place-items:center;align-content:center;gap:9px;color:${({ theme }) => theme.colors.muted};strong{color:${({ theme }) => theme.colors.text};font-size:18px;}`

export default RadioPanel
