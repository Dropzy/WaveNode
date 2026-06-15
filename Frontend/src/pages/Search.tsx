import React, { useCallback, useState, useEffect } from 'react';
import styled from 'styled-components';
import { musicAPI, SearchResult, Music, Playlist } from '../services/api';
import { useAudio } from '../contexts/AudioContext';
import { formatDuration } from '../utils/formatDuration';
import { Search as SearchIcon, Music as MusicIcon, Disc, User, ListMusic } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { TrackActionsMenu, TrackSelectionContextMenu } from '../components/TrackActionsMenu';
import { useTrackSelection } from '../hooks/useTrackSelection';

interface SearchProps {
  onTrackSelect?: (track: Music) => void;
}

const SearchContainer = styled.div`
  padding: 24px;
  overflow-y: auto;
  
  @media (max-width: 768px) {
    padding: 16px;
    padding-top: 80px; // Account for mobile menu button
  }
`;

const SearchHeader = styled.div`
  margin-bottom: 32px;
  
  @media (max-width: 768px) {
    margin-bottom: 24px;
  }
`;

const Title = styled.h1`
  color: #fff;
  font-size: 32px;
  font-weight: 700;
  margin-bottom: 24px;
  
  @media (max-width: 768px) {
    font-size: 24px;
    margin-bottom: 16px;
  }
`;

const SearchInputContainer = styled.div`
  position: relative;
  max-width: 600px;
  
  @media (max-width: 768px) {
    max-width: 100%;
  }
`;

const SearchInput = styled.input`
  width: 100%;
  padding: 16px 20px 16px 52px;
  background-color: #242424;
  border: none;
  border-radius: 24px;
  color: #fff;
  font-size: 16px;
  font-weight: 400;
  outline: none;
  transition: background-color 0.2s ease;

  &::placeholder {
    color: #b3b3b3;
  }

  &:focus {
    background-color: #2a2a2a;
  }
  
  @media (max-width: 768px) {
    padding: 14px 18px 14px 46px;
    font-size: 16px;
  }
`;

const SearchIconWrapper = styled.div`
  position: absolute;
  left: 20px;
  top: 50%;
  transform: translateY(-50%);
  color: #b3b3b3;
  pointer-events: none;
  
  @media (max-width: 768px) {
    left: 16px;
  }
`;

const LoadingIndicator = styled.div`
  position: absolute;
  right: 20px;
  top: 50%;
  transform: translateY(-50%);
  color: #1db954;
  font-size: 14px;
  font-weight: 500;
  
  @media (max-width: 768px) {
    right: 16px;
  }
`;

const ErrorMessage = styled.div`
  background-color: rgba(255, 107, 107, 0.1);
  color: #ff6b6b;
  padding: 16px;
  border-radius: 8px;
  border: 1px solid rgba(255, 107, 107, 0.3);
  margin-bottom: 24px;
  font-size: 14px;
`;

const TabsContainer = styled.div`
  margin-bottom: 24px;
`;

const TabsList = styled.div`
  display: flex;
  gap: 8px;
  border-bottom: 1px solid #282828;
  margin-bottom: 24px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  
  @media (max-width: 768px) {
    gap: 4px;
    margin-bottom: 16px;
  }
`;

const TabButton = styled.button<{ $active: boolean }>`
  background: none;
  border: none;
  color: ${props => props.$active ? '#fff' : '#b3b3b3'};
  padding: 12px 24px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  border-bottom: 2px solid ${props => props.$active ? '#1db954' : 'transparent'};
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  flex-shrink: 0;

  &:hover {
    color: #fff;
  }
  
  @media (max-width: 768px) {
    padding: 10px 16px;
    font-size: 13px;
    gap: 6px;
  }
`;

const SearchResults = styled.div`
  display: flex;
  flex-direction: column;
  gap: 32px;
  
  @media (max-width: 768px) {
    gap: 24px;
  }
`;

const SearchSection = styled.section`
  margin-bottom: 32px;
  
  @media (max-width: 768px) {
    margin-bottom: 24px;
  }
`;

const SectionTitle = styled.h3`
  color: #fff;
  font-size: 24px;
  font-weight: 700;
  margin-bottom: 20px;
  
  @media (max-width: 768px) {
    font-size: 20px;
    margin-bottom: 16px;
  }
`;

const TrackList = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
`;

const TrackItem = styled.div<{ $selected?: boolean }>`
  background-color: ${props => props.$selected ? '#3a3a3a' : '#181818'};
  border-radius: 8px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background-color: #282828;
  }
  
  @media (max-width: 768px) {
    padding: 10px 12px;
    gap: 10px;
  }
`;

const TrackCover = styled.div`
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #4a90e2, #7bb3f0);
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 16px;
  flex-shrink: 0;
  
  @media (max-width: 768px) {
    width: 35px;
    height: 35px;
    font-size: 14px;
  }
`;

const TrackInfo = styled.div`
  flex: 1;
  min-width: 0;
`;

const TrackName = styled.div`
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 13px;
  }
`;

const TrackArtist = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 11px;
  }
`;

const TrackAlbum = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 11px;
  }
`;

const TrackDuration = styled.div`
  color: #b3b3b3;
  font-size: 14px;
  font-weight: 500;
  
  @media (max-width: 768px) {
    font-size: 12px;
  }
`;

const GridContainer = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 24px;
  
  @media (max-width: 768px) {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 16px;
  }
  
  @media (max-width: 480px) {
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }
`;

const Card = styled.div`
  background-color: #181818;
  border-radius: 8px;
  padding: 16px;
  cursor: pointer;
  transition: all 0.3s ease;
  position: relative;

  &:hover {
    background-color: #282828;
    transform: translateY(-2px);
  }

  &:hover ${TrackCover} {
    background-color: #1db954;
  }
  
  @media (max-width: 768px) {
    padding: 12px;
  }
`;

const CardCover = styled.div<{ $gradient?: string }>`
  width: 100%;
  aspect-ratio: 1;
  background: ${props => props.$gradient || 'linear-gradient(135deg, #4a90e2, #7bb3f0)'};
  border-radius: 8px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 48px;
  position: relative;
  
  @media (max-width: 768px) {
    margin-bottom: 12px;
    font-size: 40px;
  }
`;

const CardInfo = styled.div`
  text-align: center;
`;

const CardTitle = styled.div`
  color: #fff;
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 13px;
  }
`;

const CardSubtitle = styled.div`
  color: #b3b3b3;
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  
  @media (max-width: 768px) {
    font-size: 11px;
  }
`;

const CardMeta = styled.div`
  color: #b3b3b3;
  font-size: 11px;
  margin-top: 4px;
  
  @media (max-width: 768px) {
    font-size: 10px;
  }
`;

const EmptyState = styled.div`
  text-align: center;
  padding: 60px 20px;
  color: #b3b3b3;
  
  @media (max-width: 768px) {
    padding: 40px 16px;
  }
`;

const EmptyStateIcon = styled.div`
  font-size: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
  
  @media (max-width: 768px) {
    font-size: 40px;
    margin-bottom: 12px;
  }
`;

const EmptyStateText = styled.div`
  font-size: 18px;
  margin-bottom: 8px;
  
  @media (max-width: 768px) {
    font-size: 16px;
  }
`;

const EmptyStateSubtext = styled.div`
  font-size: 14px;
  opacity: 0.8;
  
  @media (max-width: 768px) {
    font-size: 13px;
  }
`;

const Search: React.FC<SearchProps> = ({ onTrackSelect }) => {
  const [query, setQuery] = useState('');
  const [searchResults, setSearchResults] = useState<SearchResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'all' | 'songs' | 'albums' | 'artists' | 'playlists'>('all');
  const { playTrack } = useAudio();
  const navigate = useNavigate();
  const visibleSongs = searchResults?.songs || [];
  const trackSelection = useTrackSelection(visibleSongs);
  const [selectionMenu, setSelectionMenu] = useState<{ x: number; y: number } | null>(null);

  const handleSearch = useCallback(async () => {
    if (!query.trim()) return;

    setLoading(true);
    setError(null);

    try {
      const results = await musicAPI.comprehensiveSearch(query);
      setSearchResults(results);
    } catch (err) {
      console.error('Search error:', err);
      setError('Failed to search. Please try again.');
    } finally {
      setLoading(false);
    }
  }, [query]);

  useEffect(() => {
    const delayedSearch = setTimeout(() => {
      if (query.trim()) {
        void handleSearch();
      } else {
        setSearchResults(null);
      }
    }, 300);

    return () => clearTimeout(delayedSearch);
  }, [handleSearch, query]);

  const handleTrackPlay = (track: Music) => {
    if (onTrackSelect) {
      onTrackSelect(track);
    } else {
      playTrack(track);
    }
  };

  const handleArtistClick = (artist: { id?: string; name: string }) => {
    if (artist.id) {
      navigate(`/artist/${encodeURIComponent(artist.id)}`);
      return;
    }

    setQuery(artist.name);
    setActiveTab('songs');
  };

  const handleAlbumClick = (albumName: string) => {
    navigate(`/album/${encodeURIComponent(albumName)}`);
  };

  const handlePlaylistClick = (playlistId: string) => {
    navigate(`/playlist/${encodeURIComponent(playlistId)}`);
  };

  const renderSongs = (songs: Music[]) => (
    <SearchSection>
      <SectionTitle>Songs</SectionTitle>
      <TrackList>
        {songs.map((track, index) => (
          <TrackItem
            key={track.id}
            ref={element => { trackSelection.rowRefs.current[index] = element }}
            role="option"
            tabIndex={0}
            aria-selected={trackSelection.selectedIds.has(track.id)}
            $selected={trackSelection.selectedIds.has(track.id)}
            onClick={event => trackSelection.selectIndex(index, event)}
            onDoubleClick={() => handleTrackPlay(track)}
            onKeyDown={event => trackSelection.handleKeyDown(index, event, () => handleTrackPlay(track))}
            onContextMenu={event => {
              event.preventDefault();
              trackSelection.ensureSelected(index);
              setSelectionMenu({ x: event.clientX, y: event.clientY });
            }}
          >
            <TrackCover>
              <MusicIcon size={20} />
            </TrackCover>
            <TrackInfo>
              <TrackName>{track.title}</TrackName>
              <TrackArtist>{track.artist}</TrackArtist>
              <TrackAlbum>{track.album}</TrackAlbum>
            </TrackInfo>
            <TrackDuration>{formatDuration(track.duration)}</TrackDuration>
            <TrackActionsMenu
              track={track}
              tracks={trackSelection.selectedIds.has(track.id) ? trackSelection.selectedTracks : []}
            />
          </TrackItem>
        ))}
      </TrackList>
    </SearchSection>
  );

  const renderAlbums = (albums: Array<{ name: string; artist: string; year: number }>) => (
    <SearchSection>
      <SectionTitle>Albums</SectionTitle>
      <GridContainer>
        {albums.map((album, index) => (
          <Card key={index} onClick={() => handleAlbumClick(album.name)}>
            <CardCover $gradient="linear-gradient(135deg, #9b59b6, #c39bd3)">
              <Disc size={48} />
            </CardCover>
            <CardInfo>
              <CardTitle>{album.name}</CardTitle>
              <CardSubtitle>{album.artist}</CardSubtitle>
              <CardMeta>{album.year}</CardMeta>
            </CardInfo>
          </Card>
        ))}
      </GridContainer>
    </SearchSection>
  );

  const renderArtists = (artists: Array<{ id?: string; name: string; track_count: number; album_count: number }>) => (
    <SearchSection>
      <SectionTitle>Artists</SectionTitle>
      <GridContainer>
        {artists.map((artist) => (
          <Card key={artist.id || artist.name} onClick={() => handleArtistClick(artist)}>
            <CardCover $gradient="linear-gradient(135deg, #e74c3c, #ec7063)">
              <User size={48} />
            </CardCover>
            <CardInfo>
              <CardTitle>{artist.name}</CardTitle>
              <CardMeta>{artist.track_count} tracks • {artist.album_count} albums</CardMeta>
            </CardInfo>
          </Card>
        ))}
      </GridContainer>
    </SearchSection>
  );

  const renderPlaylists = (playlists: Playlist[]) => (
    <SearchSection>
      <SectionTitle>Playlists</SectionTitle>
      <GridContainer>
        {playlists.map((playlist) => (
          <Card key={playlist.id} onClick={() => handlePlaylistClick(playlist.id)}>
            <CardCover $gradient="linear-gradient(135deg, #3498db, #5dade2)">
              <ListMusic size={48} />
            </CardCover>
            <CardInfo>
              <CardTitle>{playlist.name}</CardTitle>
              <CardSubtitle>{playlist.description || 'No description'}</CardSubtitle>
              <CardMeta>{playlist.track_ids?.length || 0} tracks</CardMeta>
            </CardInfo>
          </Card>
        ))}
      </GridContainer>
    </SearchSection>
  );

  const getFilteredResults = () => {
    if (!searchResults) return null;

    switch (activeTab) {
      case 'songs':
        return { ...searchResults, albums: [], artists: [], playlists: [] };
      case 'albums':
        return { ...searchResults, songs: [], artists: [], playlists: [] };
      case 'artists':
        return { ...searchResults, songs: [], albums: [], playlists: [] };
      case 'playlists':
        return { ...searchResults, songs: [], albums: [], artists: [] };
      default:
        return searchResults;
    }
  };

  const filteredResults = getFilteredResults();

  const tabs = [
    { id: 'all', label: 'All', count: (searchResults?.songs?.length || 0) + (searchResults?.albums?.length || 0) + (searchResults?.artists?.length || 0) + (searchResults?.playlists?.length || 0) },
    { id: 'songs', label: 'Songs', count: searchResults?.songs?.length || 0 },
    { id: 'albums', label: 'Albums', count: searchResults?.albums?.length || 0 },
    { id: 'artists', label: 'Artists', count: searchResults?.artists?.length || 0 },
    { id: 'playlists', label: 'Playlists', count: searchResults?.playlists?.length || 0 }
  ];

  return (
    <SearchContainer>
      <SearchHeader>
        <Title>Search</Title>
        <SearchInputContainer>
          <SearchIconWrapper>
            <SearchIcon size={20} />
          </SearchIconWrapper>
          <SearchInput
            type="text"
            placeholder="Search for songs, artists, albums, playlists..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
          />
          {loading && <LoadingIndicator>Searching...</LoadingIndicator>}
        </SearchInputContainer>
      </SearchHeader>

      {error && <ErrorMessage>{error}</ErrorMessage>}

      {searchResults && (
        <TabsContainer>
          <TabsList>
            {tabs.map((tab) => (
              <TabButton
                key={tab.id}
                $active={activeTab === tab.id}
                onClick={() => setActiveTab(tab.id as typeof activeTab)}
              >
                {tab.label} ({tab.count})
              </TabButton>
            ))}
          </TabsList>
        </TabsContainer>
      )}

      <SearchResults>
        {filteredResults && (
          <>
            {filteredResults.songs && filteredResults.songs.length > 0 && renderSongs(filteredResults.songs)}
            {filteredResults.albums && filteredResults.albums.length > 0 && renderAlbums(filteredResults.albums)}
            {filteredResults.artists && filteredResults.artists.length > 0 && renderArtists(filteredResults.artists)}
            {filteredResults.playlists && filteredResults.playlists.length > 0 && renderPlaylists(filteredResults.playlists)}
            
            {(!filteredResults.songs || filteredResults.songs.length === 0) && 
             (!filteredResults.albums || filteredResults.albums.length === 0) && 
             (!filteredResults.artists || filteredResults.artists.length === 0) && 
             (!filteredResults.playlists || filteredResults.playlists.length === 0) && 
             query && !loading && (
              <EmptyState>
                <EmptyStateIcon>
                  <SearchIcon size={48} />
                </EmptyStateIcon>
                <EmptyStateText>No results found for "{query}"</EmptyStateText>
                <EmptyStateSubtext>Try different keywords or check spelling</EmptyStateSubtext>
              </EmptyState>
            )}
          </>
        )}
      </SearchResults>
      <TrackSelectionContextMenu
        tracks={trackSelection.selectedTracks}
        position={selectionMenu}
        onClose={() => setSelectionMenu(null)}
      />
    </SearchContainer>
  );
};

export { Search };
