import React, { useState } from 'react';
import { createPortal } from 'react-dom';
import styled from 'styled-components';
import { useAudio } from '../contexts/AudioContext';
import { Music } from '../services/api';
import { getTrackArtworkUrl } from '../utils/mediaUrl';
import { TrackActionsMenu } from './TrackActionsMenu';
import { 
  X, 
  Play, 
  Trash2, 
  GripVertical,
  Pause,
  Plus,
  Clock,
  Music as MusicIcon
} from 'lucide-react';

interface QueueContainerProps {
  $isOpen: boolean;
}

const QueueContainer = styled.div<QueueContainerProps>`
  position: fixed;
  right: 0;
  top: 0;
  bottom: 104px;
  width: 400px;
  background: ${({ theme }) => theme.colors.backgroundElevated};
  border-left: 1px solid ${({ theme }) => theme.colors.border};
  box-shadow: -18px 0 55px ${({ theme }) => theme.colors.shadow};
  display: flex;
  flex-direction: column;
  z-index: 1000;
  transform: translateX(${props => props.$isOpen ? '0' : '100%'});
  transition: transform 0.3s ease-in-out;
  
  @media (max-width: 768px) {
    width: 100%;
    bottom: 80px;
  }
`;

const QueueHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
  background: ${({ theme }) => theme.colors.surface};
`;

const QueueTitle = styled.h2`
  color: ${({ theme }) => theme.colors.text};
  font-size: 18px;
  font-weight: 600;
  margin: 0;
`;

const CloseButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  
  &:hover {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.controlBg};
  }
  
  svg {
    width: 20px;
    height: 20px;
  }
`;


const ClearButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  padding: 6px 12px;
  border-radius: 4px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 6px;
  
  &:hover {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.controlBg};
  }
  
  svg {
    width: 14px;
    height: 14px;
  }
`;

const QueueList = styled.div`
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
  scrollbar-color: ${({ theme }) => theme.colors.borderStrong} transparent;
`;

const QueueItem = styled.div`
  display: flex;
  align-items: center;
  padding: 8px 20px;
  cursor: pointer;
  transition: background-color 0.2s ease;
  
  &:hover {
    background: ${({ theme }) => theme.colors.controlBg};
  }
  
  &.current-track {
    background: ${({ theme }) => theme.colors.accentSoft};
    border-left: 3px solid ${({ theme }) => theme.colors.accent};
  }
`;

const DragHandle = styled.div`
  color: ${({ theme }) => theme.colors.subtle};
  cursor: grab;
  padding: 4px;
  display: flex;
  align-items: center;
  margin-right: 8px;
  
  &:active {
    cursor: grabbing;
  }
  
  svg {
    width: 16px;
    height: 16px;
  }
`;

const TrackArtwork = styled.div`
  width: 48px;
  height: 48px;
  background: ${({ theme }) => theme.colors.surfaceStrong};
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: ${({ theme }) => theme.colors.muted};
  font-size: 12px;
  margin-right: 12px;
  flex-shrink: 0;
`;

const TrackInfo = styled.div`
  flex: 1;
  min-width: 0;
`;

interface TrackNameProps {
  $isCurrent: boolean;
}

const TrackName = styled.div<TrackNameProps>`
  color: ${props => props.$isCurrent ? props.theme.colors.accent : props.theme.colors.text};
  font-size: 14px;
  font-weight: ${props => props.$isCurrent ? '600' : '400'};
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 2px;
`;

const TrackArtist = styled.div`
  color: ${({ theme }) => theme.colors.muted};
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`;

const TrackDuration = styled.div`
  color: ${({ theme }) => theme.colors.muted};
  font-size: 12px;
  margin-left: 12px;
`;

const PlayButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  padding: 8px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: 8px;
  
  &:hover {
    color: ${({ theme }) => theme.colors.text};
    background: ${({ theme }) => theme.colors.controlBg};
  }
  
  svg {
    width: 16px;
    height: 16px;
  }
`;

const RemoveButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: 4px;
  opacity: 0;
  transition: opacity 0.2s ease;
  
  ${QueueItem}:hover & {
    opacity: 1;
  }
  
  &:hover {
    color: #fff;
    background: ${({ theme }) => theme.colors.danger};
  }
  
  svg {
    width: 14px;
    height: 14px;
  }
`;

const EmptyQueue = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: ${({ theme }) => theme.colors.muted};
  text-align: center;
`;

const EmptyQueueIcon = styled.div`
  width: 64px;
  height: 64px;
  background: ${({ theme }) => theme.colors.surfaceStrong};
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  
  svg {
    width: 32px;
    height: 32px;
    color: ${({ theme }) => theme.colors.subtle};
  }
`;

const EmptyQueueText = styled.div`
  font-size: 16px;
  margin-bottom: 8px;
`;

const EmptyQueueSubtext = styled.div`
  font-size: 14px;
  color: ${({ theme }) => theme.colors.subtle};
`;

const SectionDivider = styled.div`
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
  margin: 8px 0;
`;

const SectionHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: ${({ theme }) => theme.colors.surface};
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
`;

const SectionTitle = styled.h3`
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  font-weight: 600;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
`;

const RecentlyPlayedItem = styled(QueueItem)`
  &.recently-played {
    opacity: 0.8;
    
    &:hover {
      opacity: 1;
    }
  }
`;

const AddToQueueButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.muted};
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: 8px;
  
  &:hover {
    color: ${({ theme }) => theme.colors.accent};
    background: ${({ theme }) => theme.colors.controlBg};
  }
  
  svg {
    width: 16px;
    height: 16px;
  }
`;

const NowPlayingSection = styled.div`
  padding: 16px 20px;
  background: ${({ theme }) => theme.colors.surfaceSoft};
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
`;

const NowPlayingHeader = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: ${({ theme }) => theme.colors.accent};
  font-size: 14px;
  font-weight: 600;
  
  svg {
    width: 16px;
    height: 16px;
  }
`;

const NowPlayingTrack = styled.div`
  display: flex;
  align-items: center;
  padding: 8px 0;
`;

const NowPlayingArtwork = styled.div`
  width: 56px;
  height: 56px;
  background: ${({ theme }) => theme.colors.surfaceStrong};
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  flex-shrink: 0;
  position: relative;
  
  img {
    width: 100%;
    height: 100%;
    border-radius: 4px;
    object-fit: cover;
  }
  
  span {
    color: ${({ theme }) => theme.colors.muted};
    font-size: 20px;
  }
`;

const NowPlayingInfo = styled.div`
  flex: 1;
  min-width: 0;
`;

const NowPlayingTitle = styled.div`
  color: ${({ theme }) => theme.colors.text};
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
`;

const NowPlayingArtist = styled.div`
  color: ${({ theme }) => theme.colors.muted};
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
`;

const NowPlayingDuration = styled.div`
  color: ${({ theme }) => theme.colors.muted};
  font-size: 14px;
  margin-left: 12px;
`;

const NowPlayingPlayButton = styled.button`
  background: none;
  border: none;
  color: ${({ theme }) => theme.colors.accent};
  cursor: pointer;
  padding: 8px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-left: 8px;
  
  &:hover {
    color: ${({ theme }) => theme.colors.accentHover};
    background: ${({ theme }) => theme.colors.controlBg};
  }
  
  svg {
    width: 20px;
    height: 20px;
  }
`;

const NoCurrentTrack = styled.div`
  color: ${({ theme }) => theme.colors.subtle};
  font-size: 14px;
  font-style: italic;
`;

interface QueueProps {
  isOpen: boolean;
  onClose: () => void;
}

export const Queue: React.FC<QueueProps> = ({ isOpen, onClose }) => {
  const { 
    queue, 
    currentTrackIndex, 
    isPlaying, 
    playTrackFromQueue, 
    removeFromQueue, 
    clearQueue,
    reorderQueue,
    recentlyPlayed,
    playFromQueue,
    addToQueue,
    currentTrack,
    togglePlayPause
  } = useAudio();

  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = Math.floor(seconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleDragStart = (index: number) => {
    setDraggedIndex(index);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
  };

  const handleDrop = (e: React.DragEvent, dropIndex: number) => {
    e.preventDefault();
    if (draggedIndex !== null && draggedIndex !== dropIndex) {
      reorderQueue(draggedIndex, dropIndex);
    }
    setDraggedIndex(null);
  };

  const handlePlayTrack = (index: number) => {
    playTrackFromQueue(index);
  };

  const handleRemoveTrack = (e: React.MouseEvent, index: number) => {
    e.stopPropagation();
    removeFromQueue(index);
  };

  const handleClearQueue = () => {
    if (queue.length > 0) {
      clearQueue();
    }
  };

  const handleAddToQueue = (track: Music) => {
    addToQueue(track);
  };

  const handlePlayRecentlyPlayed = (track: Music) => {
    const visibleRecentlyPlayed = recentlyPlayed.slice(0, 10);
    const index = visibleRecentlyPlayed.findIndex(item => item.id === track.id);
    playFromQueue(visibleRecentlyPlayed, index === -1 ? 0 : index);
  };

  const queueDrawer = (
    <QueueContainer $isOpen={isOpen}>
      <QueueHeader>
        <QueueTitle>Queue</QueueTitle>
        <CloseButton onClick={onClose}>
          <X />
        </CloseButton>
      </QueueHeader>
      
      <QueueList>
        {/* Now Playing Section */}
        <NowPlayingSection>
          <NowPlayingHeader>
            <MusicIcon size={16} />
            Now Playing
          </NowPlayingHeader>
          {(currentTrackIndex >= 0 && queue[currentTrackIndex]) || currentTrack ? (
            <NowPlayingTrack>
              <NowPlayingArtwork>
                {(currentTrackIndex >= 0 && queue[currentTrackIndex]) ? (
                  getTrackArtworkUrl(queue[currentTrackIndex]) ? (
                    <img 
                      src={getTrackArtworkUrl(queue[currentTrackIndex])}
                      alt={queue[currentTrackIndex].title}
                    />
                  ) : (
                    <span>♪</span>
                  )
                ) : currentTrack ? (
                  getTrackArtworkUrl(currentTrack) ? (
                    <img 
                      src={getTrackArtworkUrl(currentTrack)}
                      alt={currentTrack.title}
                    />
                  ) : (
                    <span>♪</span>
                  )
                ) : (
                  <span>♪</span>
                )}
              </NowPlayingArtwork>
              <NowPlayingInfo>
                <NowPlayingTitle>
                  {(currentTrackIndex >= 0 && queue[currentTrackIndex]) 
                    ? queue[currentTrackIndex].title 
                    : currentTrack?.title || 'Unknown Track'}
                </NowPlayingTitle>
                <NowPlayingArtist>
                  {(currentTrackIndex >= 0 && queue[currentTrackIndex]) 
                    ? `${queue[currentTrackIndex].artist} • ${queue[currentTrackIndex].album}`
                    : currentTrack ? `${currentTrack.artist} • ${currentTrack.album}` : 'Unknown Artist'}
                </NowPlayingArtist>
              </NowPlayingInfo>
              <NowPlayingDuration>
                {formatDuration(
                  (currentTrackIndex >= 0 && queue[currentTrackIndex]) 
                    ? queue[currentTrackIndex].duration 
                    : currentTrack?.duration || 0
                )}
              </NowPlayingDuration>
              <NowPlayingPlayButton onClick={togglePlayPause} title={isPlaying ? 'Pause' : 'Play'}>
                {isPlaying ? <Pause /> : <Play />}
              </NowPlayingPlayButton>
            </NowPlayingTrack>
          ) : (
            <NoCurrentTrack>No track currently playing</NoCurrentTrack>
          )}
        </NowPlayingSection>

        {/* Next Up Section */}
        {queue.length > 0 && (
          <>
            <SectionHeader>
              <SectionTitle>
                Next Up ({queue.length - (currentTrackIndex >= 0 ? 1 : 0)} tracks)
              </SectionTitle>
              <ClearButton onClick={handleClearQueue}>
                <Trash2 />
                Clear
              </ClearButton>
            </SectionHeader>
            
            {queue.map((track, index) => (
              index !== currentTrackIndex && (
                <QueueItem
                  key={`queue-${track.id}-${index}`}
                  onClick={() => handlePlayTrack(index)}
                  draggable
                  onDragStart={() => handleDragStart(index)}
                  onDragOver={handleDragOver}
                  onDrop={(e) => handleDrop(e, index)}
                >
                  <DragHandle>
                    <GripVertical />
                  </DragHandle>
                  
                  <TrackArtwork>
                    {getTrackArtworkUrl(track) ? (
                      <img 
                        src={getTrackArtworkUrl(track)}
                        alt={track.title}
                        style={{ width: '100%', height: '100%', borderRadius: '4px', objectFit: 'cover' }}
                      />
                    ) : (
                      <span>♪</span>
                    )}
                  </TrackArtwork>
                  
                  <TrackInfo>
                    <TrackName $isCurrent={false}>
                      {track.title}
                    </TrackName>
                    <TrackArtist>
                      {track.artist} • {track.album}
                    </TrackArtist>
                  </TrackInfo>
                  
                  <TrackDuration>
                    {formatDuration(track.duration)}
                  </TrackDuration>
                  
                  <PlayButton>
                    <Play />
                  </PlayButton>
                  <TrackActionsMenu track={track} />
                  
                  <RemoveButton onClick={(e) => handleRemoveTrack(e, index)}>
                    <X />
                  </RemoveButton>
                </QueueItem>
              )
            ))}
          </>
        )}
        
        {/* Empty State */}
        {queue.length === 0 && (!recentlyPlayed || recentlyPlayed.length === 0) && (
          <EmptyQueue>
            <EmptyQueueIcon>
              <Plus />
            </EmptyQueueIcon>
            <EmptyQueueText>Your queue is empty</EmptyQueueText>
            <EmptyQueueSubtext>Add tracks to see them here</EmptyQueueSubtext>
          </EmptyQueue>
        )}
        
        {/* Recently Played Section */}
        {recentlyPlayed && recentlyPlayed.length > 0 && (
          <>
            <SectionDivider />
            <SectionHeader>
              <SectionTitle>
                <Clock size={16} />
                Recently Played
              </SectionTitle>
            </SectionHeader>
            
            {recentlyPlayed.slice(0, 10).map((track, index) => (
              <RecentlyPlayedItem
                key={`recent-${track.id}-${index}`}
                className="recently-played"
                onClick={() => handlePlayRecentlyPlayed(track)}
              >
                <TrackArtwork>
                  {getTrackArtworkUrl(track) ? (
                    <img 
                      src={getTrackArtworkUrl(track)}
                      alt={track.title}
                      style={{ width: '100%', height: '100%', borderRadius: '4px', objectFit: 'cover' }}
                    />
                  ) : (
                    <span>♪</span>
                  )}
                </TrackArtwork>
                
                <TrackInfo>
                  <TrackName $isCurrent={false}>
                    {track.title}
                  </TrackName>
                  <TrackArtist>
                    {track.artist} • {track.album}
                  </TrackArtist>
                </TrackInfo>
                
                <TrackDuration>
                  {formatDuration(track.duration)}
                </TrackDuration>
                
                <PlayButton>
                  <Play />
                </PlayButton>
                <TrackActionsMenu track={track} />
                
                <AddToQueueButton onClick={(e) => {
                  e.stopPropagation();
                  handleAddToQueue(track);
                }}>
                  <Plus />
                </AddToQueueButton>
              </RecentlyPlayedItem>
            ))}
          </>
        )}
      </QueueList>
    </QueueContainer>
  );

  return createPortal(queueDrawer, document.body);
};
