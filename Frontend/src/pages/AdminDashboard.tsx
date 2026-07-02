import React, { useCallback, useEffect, useMemo, useState } from 'react'
import styled, { useTheme } from 'styled-components'
import {
  Activity,
  AlertTriangle,
  ArrowUp,
  CheckCircle2,
  ChevronRight,
  Clock3,
  Database,
  Disc3,
  FolderOpen,
  HardDrive,
  Image,
  KeyRound,
  LayoutDashboard,
  ListMusic,
  Music,
  PackagePlus,
  Plug,
  Power,
  RefreshCw,
  ScanLine,
  Shield,
  Square,
  Sparkles,
  Tags,
  Trash2,
  Upload,
  UserPlus,
  UserRound,
  Users,
  Wifi,
  X,
  XCircle,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { adminArtistImagesAPI, adminIntegrationsAPI, adminUpdateAPI, api, ArtistImage, PluginRecord, type LastFMIntegrationSettings, type UpdateStatus, User } from '../services/api'
import websocketService, { ScanStatus as WebSocketScanStatus } from '../services/websocket'
import { getArtworkGradient, resolveMediaUrl } from '../utils/mediaUrl'

type AdminTab = 'overview' | 'library' | 'users' | 'enrichment' | 'plugins' | 'integrations' | 'system'
type JobAction = 'library' | 'metadata' | 'artwork' | null

interface AdminStats {
  total_tracks: number
  total_albums: number
  total_artists: number
  total_playlists: number
  connected_users: number
  enrichment: {
    total_tracks: number
    total_artists: number
    tracks_with_metadata: number
    tracks_with_cover_art: number
    artists_with_images: number
  }
}

interface ScanStatus {
  id: string
  type: string
  status: 'pending' | 'running' | 'stopping' | 'completed' | 'completed_with_errors' | 'failed' | 'stopped'
  progress: number
  total_files: number
  processed: number
  current_file?: string
  errors?: string[]
  songs_added: number
  songs_updated: number
  tracks_skipped: number
  started_at?: string
  completed_at?: string
}

interface MusicSource {
  id: string
  path: string
  created_at: string
}

interface AutomaticUpdateSettings {
  enabled: boolean
  interval_minutes: number
  last_checked_at?: string
  last_scan_at?: string
  last_reason?: string
  last_error?: string
}

interface DirectoryBrowserData {
  current_path: string
  parent_path: string
  directories: Array<{
    name: string
    path: string
  }>
  roots: string[]
}

interface AdminArtist {
  id: string
  name: string
  track_count: number
  album_count: number
  image_url?: string
  image_small_url?: string
  image_medium_url?: string
  image_large_url?: string
}

interface SystemStatus {
  version: string
  uptime_seconds: number
  go_version: string
  goroutines: number
  active_streams: number
  database_open: number
  database_in_use: number
  database_idle: number
  artwork_bytes: number
  artwork_files: number
  automatic_updates: boolean
}

interface SourceDiagnostic {
  path: string
  accessible: boolean
  error?: string
  total_bytes: number
  free_bytes: number
  used_bytes: number
  used_percent: number
  space_status: 'healthy' | 'warning' | 'critical' | 'unknown' | 'unavailable'
}

interface TrackDiagnostic {
  id: string
  title: string
  artist: string
  album: string
  file_path: string
  format: string
  issue: string
}

interface DuplicateDiagnostic {
  title: string
  artist: string
  count: number
  track_ids: string[]
  paths: string[]
}

interface LibraryDiagnostics {
  indexed_tracks: number
  healthy_tracks: number
  health_score: number
  issue_count: number
  missing_files: number
  duplicate_groups: number
  invalid_metadata: number
  unsupported_formats: number
  missing_artwork: number
  unavailable_sources: number
  low_space_sources: number
  sources: SourceDiagnostic[]
  missing_file_details: TrackDiagnostic[]
  invalid_metadata_details: TrackDiagnostic[]
  unsupported_format_details: TrackDiagnostic[]
  missing_artwork_details: TrackDiagnostic[]
  duplicate_details: DuplicateDiagnostic[]
  details_truncated: boolean
  generated_at: string
}

interface ApiError {
  response?: {
    data?: {
      error?: string
    }
  }
  message?: string
}

const Container = styled.div`
  padding: 28px clamp(16px, 2.5vw, 36px) 48px;
  min-height: 100%;

  @media (max-width: 768px) {
    padding-top: 80px;
  }
`

const Header = styled.header`
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 24px;

  @media (max-width: 720px) {
    flex-direction: column;
  }
`

const Title = styled.h1`
  margin: 0 0 6px;
  font-size: clamp(28px, 4vw, 38px);
`

const Subtitle = styled.p`
  color: #c7c7c7;
  line-height: 1.5;
`

const LastUpdated = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  color: #c0c0c0;
  font-size: 13px;
  white-space: nowrap;
  padding-top: 10px;
`

const Notice = styled.div<{ $type: 'success' | 'error' }>`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
  padding: 13px 16px;
  border: 1px solid ${props => props.$type === 'success' ? props.theme.colors.borderStrong : 'rgba(255, 107, 107, .45)'};
  border-radius: 10px;
  background: ${props => props.$type === 'success' ? props.theme.colors.accentSoft : 'rgba(255, 107, 107, .12)'};
  color: ${props => props.$type === 'success' ? props.theme.colors.accent : '#ff9292'};
`

const DismissButton = styled.button`
  display: grid;
  place-items: center;
  color: inherit;
  padding: 4px;
`

const Tabs = styled.nav`
  display: flex;
  gap: 8px;
  padding: 5px;
  margin-bottom: 24px;
  border-radius: 12px;
  background: rgba(18, 18, 18, 0.7);
  overflow-x: auto;
`

const TabButton = styled.button<{ $active: boolean }>`
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 42px;
  padding: 10px 16px;
  border-radius: 8px;
  color: ${props => props.$active ? '#fff' : '#b3b3b3'};
  background: ${props => props.$active ? '#2b2b2b' : 'transparent'};
  font-weight: 700;
  white-space: nowrap;

  &:hover {
    color: #fff;
    background: #252525;
  }
`

const Panel = styled.section`
  border: 1px solid #2c2c2c;
  border-radius: 14px;
  background: #171717;
  overflow: hidden;
`

const PanelHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
  border-bottom: 1px solid #2c2c2c;

  @media (max-width: 640px) {
    align-items: flex-start;
    flex-direction: column;
  }
`

const PanelTitle = styled.h2`
  margin: 0 0 4px;
  font-size: 20px;
`

const PanelDescription = styled.p`
  color: #a9a9a9;
  font-size: 14px;
  line-height: 1.45;
`

const PanelBody = styled.div`
  padding: 20px;
`

const StatsGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(5, minmax(130px, 1fr));
  gap: 14px;
  margin-bottom: 20px;

  @media (max-width: 1050px) {
    grid-template-columns: repeat(3, 1fr);
  }

  @media (max-width: 600px) {
    grid-template-columns: repeat(2, 1fr);
  }
`

const StatCard = styled.div`
  padding: 18px;
  min-height: 125px;
  border: 1px solid #303030;
  border-radius: 12px;
  background: linear-gradient(145deg, #202020, #181818);
`

const StatIcon = styled.div`
  color: ${({ theme }) => theme.colors.accent};
  margin-bottom: 18px;
`

const StatValue = styled.div`
  font-size: 28px;
  font-weight: 800;
`

const StatLabel = styled.div`
  margin-top: 3px;
  color: #aaa;
  font-size: 13px;
`

const TwoColumn = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(300px, .75fr);
  gap: 20px;

  @media (max-width: 900px) {
    grid-template-columns: 1fr;
  }
`

const ActionGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(230px, 1fr));
  gap: 16px;
`

const ActionCard = styled.div`
  display: flex;
  flex-direction: column;
  min-height: 230px;
  padding: 20px;
  border: 1px solid #303030;
  border-radius: 12px;
  background: #1d1d1d;
`

const ActionIcon = styled.div`
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  margin-bottom: 18px;
  border-radius: 10px;
  color: ${({ theme }) => theme.colors.accent};
  background: ${({ theme }) => theme.colors.accentSoft};
`

const ActionTitle = styled.h3`
  margin: 0 0 8px;
  font-size: 17px;
`

const ActionText = styled.p`
  flex: 1;
  color: #aaa;
  font-size: 14px;
  line-height: 1.5;
  margin-bottom: 18px;
`

const Button = styled.button<{ $variant?: 'primary' | 'danger' | 'secondary' }>`
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 40px;
  padding: 9px 15px;
  border: 1px solid ${props => props.$variant === 'danger' ? '#7f3039' : props.$variant === 'primary' ? props.theme.colors.accent : props.theme.colors.borderStrong};
  border-radius: 999px;
  background: ${props => props.$variant === 'danger' ? '#722c34' : props.$variant === 'primary' ? props.theme.colors.accentGradient : 'transparent'};
  color: ${props => props.$variant === 'primary' ? props.theme.colors.accentText : props.theme.colors.text};
  font-weight: 750;

  &:hover:not(:disabled) {
    filter: brightness(1.12);
    transform: translateY(-1px);
  }

  &:disabled {
    opacity: .5;
    cursor: not-allowed;
  }
`

const ButtonRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
`

const LibraryStack = styled.div`
  display: grid;
  gap: 20px;
`

const SourceHelp = styled.p`
  margin: 0 0 18px;
  color: #999;
  font-size: 13px;
  line-height: 1.5;
`

const PickerBackdrop = styled.div`
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgba(0, 0, 0, .72);
  backdrop-filter: blur(6px);
`

const PickerDialog = styled.div`
  width: min(680px, 100%);
  max-height: min(720px, calc(100vh - 40px));
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid #3a3a3a;
  border-radius: 14px;
  background: #181818;
  box-shadow: 0 24px 80px rgba(0, 0, 0, .55);
`

const PickerHeader = styled.div`
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 20px;
  border-bottom: 1px solid #303030;
`

const PickerPath = styled.div`
  padding: 11px 14px;
  overflow-x: auto;
  border-bottom: 1px solid #303030;
  color: #ccc;
  background: #202020;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 13px;
  white-space: nowrap;
`

const PickerToolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px 16px;
  border-bottom: 1px solid #303030;
`

const DirectoryList = styled.div`
  min-height: 220px;
  overflow-y: auto;
  padding: 8px;
`

const DirectoryButton = styled.button`
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 12px;
  border-radius: 8px;
  color: #eee;
  text-align: left;

  &:hover {
    background: #292929;
  }

  span {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
`

const PickerFooter = styled.div`
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
  padding: 16px 20px;
  border-top: 1px solid #303030;
`

const SourceList = styled.div`
  display: grid;
  gap: 10px;
`

const SourceRow = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 13px 14px;
  border: 1px solid #303030;
  border-radius: 10px;
  background: #1d1d1d;

  @media (max-width: 680px) {
    align-items: flex-start;
    flex-direction: column;
  }
`

const SourcePath = styled.div`
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  color: #ddd;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 13px;

  span {
    overflow-wrap: anywhere;
  }
`

const HealthOverview = styled.div`
  display: grid;
  grid-template-columns: minmax(220px, .55fr) minmax(0, 1.45fr);
  gap: 20px;

  @media (max-width: 780px) {
    grid-template-columns: 1fr;
  }
`

const HealthScoreCard = styled.div<{ $score: number }>`
  display: grid;
  align-content: center;
  justify-items: center;
  min-height: 220px;
  padding: 24px;
  border: 1px solid ${props => props.$score >= 90 ? props.theme.colors.borderStrong : props.$score >= 70 ? 'rgba(255, 193, 7, .38)' : 'rgba(255, 107, 107, .4)'};
  border-radius: 12px;
  text-align: center;
  background: ${props => props.$score >= 90 ? props.theme.colors.accentSoft : props.$score >= 70 ? 'rgba(255, 193, 7, .08)' : 'rgba(255, 107, 107, .08)'};
`

const HealthScoreValue = styled.div`
  font-size: clamp(48px, 7vw, 72px);
  font-weight: 850;
  line-height: 1;
`

const HealthScoreLabel = styled.div`
  margin-top: 10px;
  font-size: 16px;
  font-weight: 750;
`

const DiagnosticGrid = styled.div`
  display: grid;
  grid-template-columns: repeat(3, minmax(130px, 1fr));
  gap: 12px;

  @media (max-width: 580px) {
    grid-template-columns: repeat(2, minmax(120px, 1fr));
  }
`

const DiagnosticCard = styled.div<{ $problem?: boolean }>`
  padding: 16px;
  border: 1px solid ${props => props.$problem ? 'rgba(255, 193, 7, .28)' : '#303030'};
  border-radius: 10px;
  background: ${props => props.$problem ? 'rgba(255, 193, 7, .055)' : '#1d1d1d'};
`

const DiagnosticValue = styled.div`
  font-size: 24px;
  font-weight: 850;
`

const DiagnosticLabel = styled.div`
  margin-top: 5px;
  color: #aaa;
  font-size: 12px;
  line-height: 1.35;
`

const StorageDetails = styled.div`
  flex: 1;
  min-width: 0;
`

const StorageHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
`

const StorageProgress = styled.div`
  height: 7px;
  overflow: hidden;
  border-radius: 999px;
  background: #353535;
`

const StorageProgressFill = styled.div<{ $percent: number; $status: SourceDiagnostic['space_status'] }>`
  width: ${props => Math.min(100, Math.max(0, props.$percent))}%;
  height: 100%;
  border-radius: inherit;
  background: ${props => props.$status === 'critical' ? '#ff6b6b' : props.$status === 'warning' ? '#ffc107' : props.theme.colors.accentGradient};
`

const IssueSections = styled.div`
  display: grid;
  gap: 14px;
`

const IssueGroup = styled.details`
  border: 1px solid #303030;
  border-radius: 10px;
  background: #1b1b1b;
  overflow: hidden;

  summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 15px 16px;
    cursor: pointer;
    font-weight: 750;
    list-style: none;
  }

  summary::-webkit-details-marker {
    display: none;
  }
`

const IssueList = styled.div`
  display: grid;
  max-height: 360px;
  overflow-y: auto;
  border-top: 1px solid #303030;
`

const IssueRow = styled.div`
  display: grid;
  grid-template-columns: minmax(150px, .75fr) minmax(220px, 1.25fr);
  gap: 16px;
  padding: 13px 16px;
  border-bottom: 1px solid #292929;

  &:last-child {
    border-bottom: 0;
  }

  @media (max-width: 680px) {
    grid-template-columns: 1fr;
    gap: 5px;
  }
`

const IssuePath = styled.div`
  color: #999;
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px;
  overflow-wrap: anywhere;
`

const LogPanel = styled.div`
  margin: 16px 0 0;
  max-height: 260px;
  overflow: auto;
  padding: 14px;
  border: 1px solid ${({ theme }) => theme.colors.border};
  border-radius: 10px;
  background: ${({ theme }) => theme.colors.backgroundElevated};
  color: ${({ theme }) => theme.colors.muted};
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 12px;
  line-height: 1.45;
  white-space: pre-wrap;
`

const ArtistImageList = styled.div`
  display: grid;
  gap: 10px;
  max-height: 520px;
  overflow-y: auto;
`

const ArtistImageRow = styled.div`
  display: grid;
  grid-template-columns: 58px minmax(160px, 1fr) auto;
  align-items: center;
  gap: 14px;
  padding: 11px;
  border: 1px solid #303030;
  border-radius: 10px;
  background: #1d1d1d;

  @media (max-width: 680px) {
    grid-template-columns: 52px 1fr;

    ${ButtonRow} {
      grid-column: 1 / -1;
    }
  }
`

const ArtistImagePreview = styled.div<{ $imageUrl?: string; $fallback: string }>`
  width: 58px;
  height: 58px;
  display: grid;
  place-items: center;
  overflow: hidden;
  border-radius: 50%;
  background: ${props => props.$imageUrl ? `url("${props.$imageUrl}")` : props.$fallback};
  background-size: cover;
  background-position: center;
  color: #fff;
`

const CandidateList = styled.div`
  grid-column: 1 / -1;
  display: grid;
  gap: 8px;
  padding: 8px 0 0 72px;

  @media (max-width: 680px) {
    padding-left: 0;
  }
`

const CandidateRow = styled.div`
  display: grid;
  grid-template-columns: 74px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 10px;
  border: 1px solid #303030;
  border-radius: 8px;
  background: #181818;

  @media (max-width: 680px) {
    grid-template-columns: 58px minmax(0, 1fr);

    ${ButtonRow} {
      grid-column: 1 / -1;
    }
  }
`

const CandidateImage = styled.img`
  width: 74px;
  height: 74px;
  object-fit: cover;
  border-radius: 8px;
`

const HiddenFileInput = styled.input`
  display: none;
`

const DangerZone = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin-top: 20px;
  padding: 16px;
  border: 1px solid rgba(255, 107, 107, .35);
  border-radius: 11px;
  background: rgba(255, 107, 107, .07);

  @media (max-width: 680px) {
    align-items: flex-start;
    flex-direction: column;
  }
`

const ProgressTrack = styled.div`
  height: 8px;
  margin: 14px 0 8px;
  overflow: hidden;
  border-radius: 999px;
  background: #343434;
`

const ProgressFill = styled.div<{ $progress: number }>`
  width: ${props => Math.min(100, Math.max(0, props.$progress))}%;
  height: 100%;
  border-radius: inherit;
  background: ${({ theme }) => theme.colors.accentGradient};
  transition: width .25s ease;
`

const CurrentJob = styled.div`
  padding: 18px;
  border: 1px solid ${({ theme }) => theme.colors.borderStrong};
  border-radius: 12px;
  background: ${({ theme }) => theme.colors.accentSoft};
`

const JobHeader = styled.div`
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
`

const JobFile = styled.div`
  margin-top: 8px;
  color: #aaa;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const ErrorSummary = styled.div`
  display: grid;
  gap: 6px;
  margin-top: 12px;
  padding: 10px 12px;
  border: 1px solid rgba(255, 107, 107, .3);
  border-radius: 8px;
  color: #ffaaaa;
  background: rgba(255, 107, 107, .08);
  font-size: 12px;
`

const StatusBadge = styled.span<{ $status: string }>`
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 9px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 800;
  text-transform: capitalize;
  color: ${props => props.$status.startsWith('completed') ? props.theme.colors.accent : props.$status === 'failed' ? '#ff8d8d' : '#ffd56a'};
  background: ${props => props.$status.startsWith('completed') ? props.theme.colors.accentSoft : props.$status === 'failed' ? 'rgba(255, 107, 107, .13)' : 'rgba(255, 193, 7, .13)'};
`

const HistoryList = styled.div`
  display: grid;
  gap: 10px;
`

const HistoryItem = styled.div`
  display: grid;
  grid-template-columns: minmax(160px, 1fr) minmax(120px, .55fr) minmax(120px, .55fr) auto;
  align-items: center;
  gap: 14px;
  padding: 14px;
  border: 1px solid #2d2d2d;
  border-radius: 10px;
  background: #1c1c1c;

  @media (max-width: 720px) {
    grid-template-columns: 1fr 1fr;
  }
`

const PrimaryText = styled.div`
  font-weight: 700;
`

const SecondaryText = styled.div`
  margin-top: 3px;
  color: #999;
  font-size: 12px;
`

const CoverageRow = styled.div`
  margin-bottom: 18px;
`

const CoverageHeader = styled.div`
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 14px;
`

const TableWrap = styled.div`
  overflow-x: auto;
`

const Table = styled.table`
  width: 100%;
  border-collapse: collapse;
  min-width: 720px;

  th,
  td {
    padding: 14px 16px;
    text-align: left;
    border-bottom: 1px solid #2c2c2c;
  }

  th {
    color: #a7a7a7;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: .04em;
    background: #1d1d1d;
  }

  tr:hover td {
    background: #202020;
  }
`

const Select = styled.select`
  padding: 8px 10px;
  border: 1px solid #4a4a4a;
  border-radius: 7px;
  color: #fff;
  background: #282828;

  &:disabled {
    opacity: .5;
  }
`

const AccessControl = styled.div`
	display: grid;
	gap: 7px;
	min-width: 220px;

	label {
		display: flex;
		align-items: center;
		gap: 7px;
		font-size: 13px;
	}
`

const SourceChoices = styled.div`
	display: grid;
	gap: 5px;
	padding-left: 4px;
	color: #bbb;

	label {
		font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
		font-size: 11px;
	}
`

const AutomationRow = styled.div`
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(150px, 220px) auto;
  align-items: center;
  gap: 16px;
  padding: 16px;
  border: 1px solid #333;
  border-radius: 10px;
  background: #191919;

  label {
    display: flex;
    align-items: center;
    gap: 10px;
    color: #fff;
    font-weight: 600;
  }

  input[type='number'] {
    width: 100%;
    padding: 9px 10px;
    border: 1px solid #4a4a4a;
    border-radius: 7px;
    color: #fff;
    background: #282828;
  }

  @media (max-width: 760px) {
    grid-template-columns: 1fr;
  }
`

const UserForm = styled.form`
  display: grid;
  grid-template-columns: repeat(4, minmax(140px, 1fr)) auto;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid #2c2c2c;
  background: #181818;

  input,
  select {
    min-width: 0;
    padding: 10px 12px;
    border: 1px solid #454545;
    border-radius: 8px;
    color: #fff;
    background: #242424;
  }

  @media (max-width: 920px) {
    grid-template-columns: 1fr 1fr;
  }
`

const PluginForm = styled.form`
  display: grid;
  gap: 14px;
  padding: 18px;
  border-bottom: 1px solid #2c2c2c;
  background: #181818;

  textarea {
    width: 100%;
    min-height: 320px;
    resize: vertical;
    padding: 14px;
    border: 1px solid #454545;
    border-radius: 9px;
    color: #e8e8e8;
    background: #101010;
    font: 13px/1.55 ui-monospace, SFMono-Regular, Consolas, monospace;
  }
`

const IntegrationForm = styled.div`
  display: grid;
  gap: 16px;
  max-width: 760px;

  label {
    display: grid;
    gap: 7px;
    color: #b7b7b7;
    font-size: 14px;
  }

  input {
    width: 100%;
    padding: 11px 12px;
    border: 1px solid #454545;
    border-radius: 8px;
    color: #fff;
    background: #242424;
  }
`

const PluginMeta = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-top: 5px;
  color: #999;
  font-size: 12px;
`

const EmptyState = styled.div`
  padding: 42px 20px;
  color: #aaa;
  text-align: center;
`

const EmptyIcon = styled.div`
  margin-bottom: 12px;
  color: #666;
`

const AccessDenied = styled.div`
  display: grid;
  place-items: center;
  min-height: 480px;
  text-align: center;
`

const Spin = styled(RefreshCw)<{ $active?: boolean }>`
  animation: ${props => props.$active ? 'spin 1s linear infinite' : 'none'};

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
`

const formatNumber = (value?: number) => new Intl.NumberFormat().format(value || 0)

const formatVersion = (value?: string | null) => {
  const version = value?.trim().replace(/^v+/i, '')
  return version ? `v${version}` : '-'
}

const formatBytes = (value: number) => {
  if (!value) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const unitIndex = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / (1024 ** unitIndex)).toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}

const formatDuration = (seconds: number) => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days) return `${days}d ${hours}h`
  if (hours) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

const getErrorMessage = (error: unknown, fallback: string) => {
  const apiError = error as ApiError
  return apiError.response?.data?.error || apiError.message || fallback
}

const formatDate = (value?: string) => {
  if (!value) return 'Not available'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? 'Not available' : date.toLocaleString()
}

const scanLabel = (type: string) => {
  if (type === 'library') return 'Library scan'
  if (type === 'cover-art' || type === 'cover-art-enrichment') return 'Cover art enrichment'
  if (type === 'artist-enrichment') return 'Metadata enrichment'
  if (type === 'artist-image-discovery') return 'Local artist image discovery'
  return type.replace(/-/g, ' ')
}

const scanResultText = (scan: ScanStatus) => {
  if (scan.type === 'artist-image-discovery') {
    return `${scan.songs_added || 0} folder · ${scan.songs_updated || 0} embedded · ${scan.tracks_skipped || 0} unchanged`
  }
  return `${scan.songs_added || 0} added · ${scan.songs_updated || 0} updated`
}

const downloadPluginTemplate = JSON.stringify({
  schema_version: 1,
  id: 'mobile.downloads',
  name: 'Mobile Downloads',
  version: '1.0.0',
  description: 'Adds a Download to Device action to track menus.',
  permissions: ['download'],
  contributes: {
    track_actions: [{
      id: 'download-to-device',
      label: 'Download to Device',
      icon: 'download',
      action_type: 'download',
      url: '/api/music/{id}/download',
    }],
  },
}, null, 2)

const getScanProgress = (scan: ScanStatus) => {
  if (scan.status === 'completed' || scan.status === 'completed_with_errors') {
    return 100
  }

  if (scan.total_files > 0) {
    return Math.min(100, Math.max(0, Math.round((scan.processed / scan.total_files) * 100)))
  }

  return Math.min(100, Math.max(0, scan.progress || 0))
}

const statusIcon = (status: string) => {
  if (status === 'completed' || status === 'completed_with_errors') return <CheckCircle2 size={14} />
  if (status === 'failed') return <XCircle size={14} />
  return <Clock3 size={14} />
}

const AdminDashboard: React.FC = () => {
  const { user } = useAuth()
  const theme = useTheme()
  const [activeTab, setActiveTab] = useState<AdminTab>('overview')
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [users, setUsers] = useState<User[]>([])
  const [scans, setScans] = useState<ScanStatus[]>([])
  const [musicSources, setMusicSources] = useState<MusicSource[]>([])
  const [showUserForm, setShowUserForm] = useState(false)
  const [newUsername, setNewUsername] = useState('')
  const [newEmail, setNewEmail] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [newRole, setNewRole] = useState<User['role']>('user')
  const [creatingUser, setCreatingUser] = useState(false)
	const [userAccessAction, setUserAccessAction] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [jobAction, setJobAction] = useState<JobAction>(null)
  const [stoppingLibraryScan, setStoppingLibraryScan] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null)
  const [sourceAction, setSourceAction] = useState<string | null>(null)
  const [clearingLibrary, setClearingLibrary] = useState(false)
  const [directoryPicker, setDirectoryPicker] = useState<DirectoryBrowserData | null>(null)
  const [directoryPickerLoading, setDirectoryPickerLoading] = useState(false)
  const [artists, setArtists] = useState<AdminArtist[]>([])
  const [artistImageAction, setArtistImageAction] = useState<string | null>(null)
  const [discoveringArtistImages, setDiscoveringArtistImages] = useState(false)
  const [candidateArtistId, setCandidateArtistId] = useState<string | null>(null)
  const [artistImageCandidates, setArtistImageCandidates] = useState<Record<string, ArtistImage[]>>({})
  const [automaticUpdates, setAutomaticUpdates] = useState<AutomaticUpdateSettings>({
    enabled: false,
    interval_minutes: 60,
  })
  const [savingAutomaticUpdates, setSavingAutomaticUpdates] = useState(false)
  const [backupAction, setBackupAction] = useState<'download' | 'restore' | null>(null)
  const [systemStatus, setSystemStatus] = useState<SystemStatus | null>(null)
  const [diagnostics, setDiagnostics] = useState<LibraryDiagnostics | null>(null)
  const [plugins, setPlugins] = useState<PluginRecord[]>([])
  const [showPluginForm, setShowPluginForm] = useState(false)
  const [pluginManifest, setPluginManifest] = useState(downloadPluginTemplate)
  const [pluginAction, setPluginAction] = useState<string | null>(null)
  const [lastFMIntegration, setLastFMIntegration] = useState<LastFMIntegrationSettings>({
    api_key: '',
    shared_secret: '',
    has_shared_secret: false,
    configured: false,
  })
  const [savingLastFMIntegration, setSavingLastFMIntegration] = useState(false)
  const [updateStatus, setUpdateStatus] = useState<UpdateStatus | null>(null)
  const [updateAction, setUpdateAction] = useState<'check' | 'run' | null>(null)

  const runningScan = useMemo(
    () => scans.find(scan => scan.status === 'running' || scan.status === 'pending' || scan.status === 'stopping'),
    [scans],
  )

  const completedScans = useMemo(
    () => scans.filter(scan => scan.status !== 'running' && scan.status !== 'pending' && scan.status !== 'stopping'),
    [scans],
  )

  const artistImageScan = useMemo(
    () => scans.find(scan => scan.type === 'artist-image-discovery'),
    [scans],
  )

  const loadDashboard = useCallback(async (quiet = false, includeDiagnostics = true) => {
    if (!quiet) setLoading(true)
    setRefreshing(true)

    try {
      const [
        statsResponse,
        usersResponse,
        scansResponse,
        sourcesResponse,
        artistsResponse,
        automaticUpdatesResponse,
        systemResponse,
        diagnosticsResponse,
        pluginsResponse,
        lastFMIntegrationResponse,
        updateStatusResponse,
      ] = await Promise.all([
        api.get('/admin/stats'),
        api.get('/admin/users'),
        api.get('/admin/scans'),
        api.get('/admin/library/sources'),
        api.get('/artists'),
        api.get('/admin/library/automatic-updates'),
        api.get('/admin/system/status'),
        includeDiagnostics ? api.get('/admin/library/diagnostics') : Promise.resolve(null),
        api.get('/admin/plugins'),
        adminIntegrationsAPI.getLastFM(),
        adminUpdateAPI.getStatus(),
      ])

      setStats(statsResponse.data?.data || null)
      setUsers(usersResponse.data?.data || [])
      setScans(scansResponse.data?.data || [])
      setMusicSources(sourcesResponse.data?.data || [])
      setArtists(artistsResponse.data?.data || [])
      setAutomaticUpdates(automaticUpdatesResponse.data?.data || { enabled: false, interval_minutes: 60 })
      setSystemStatus(systemResponse.data?.data || null)
      if (diagnosticsResponse) {
        setDiagnostics(diagnosticsResponse.data?.data || null)
      }
      setPlugins(pluginsResponse.data?.data || [])
      setLastFMIntegration({
        ...lastFMIntegrationResponse,
        shared_secret: '',
      })
      setUpdateStatus(updateStatusResponse)
      setLastUpdated(new Date())
      setError(null)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not refresh the administration dashboard'))
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [])

  const installPlugin = async (event: React.FormEvent) => {
    event.preventDefault()
    try {
      setPluginAction('install')
      setError(null)
      const manifest = JSON.parse(pluginManifest)
      await api.post('/admin/plugins', manifest)
      setSuccess('Plugin installed and enabled. Its home rows are now available.')
      setShowPluginForm(false)
      await loadDashboard(true)
    } catch (requestError) {
      if (requestError instanceof SyntaxError) {
        setError(`Plugin manifest is not valid JSON: ${requestError.message}`)
      } else {
        setError(getErrorMessage(requestError, 'Could not install the plugin'))
      }
    } finally {
      setPluginAction(null)
    }
  }

  const editPlugin = (plugin: PluginRecord) => {
    setPluginManifest(JSON.stringify(plugin.manifest, null, 2))
    setShowPluginForm(true)
    setError(null)
  }

  const setPluginEnabled = async (plugin: PluginRecord, enabled: boolean) => {
    try {
      setPluginAction(plugin.id)
      setError(null)
      await api.put(`/admin/plugins/${encodeURIComponent(plugin.id)}/enabled`, { enabled })
      setPlugins(current => current.map(item => item.id === plugin.id ? { ...item, enabled } : item))
      setSuccess(`${plugin.name} ${enabled ? 'enabled' : 'disabled'}`)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not update the plugin'))
    } finally {
      setPluginAction(null)
    }
  }

  const removePlugin = async (plugin: PluginRecord) => {
    if (!window.confirm(`Remove ${plugin.name}? Its contributed sections will disappear immediately.`)) return
    try {
      setPluginAction(plugin.id)
      setError(null)
      await api.delete(`/admin/plugins/${encodeURIComponent(plugin.id)}`)
      setPlugins(current => current.filter(item => item.id !== plugin.id))
      setSuccess(`${plugin.name} removed`)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not remove the plugin'))
    } finally {
      setPluginAction(null)
    }
  }

  const saveLastFMIntegration = async () => {
    try {
      setSavingLastFMIntegration(true)
      setError(null)
      const saved = await adminIntegrationsAPI.saveLastFM(lastFMIntegration)
      setLastFMIntegration({
        ...saved,
        shared_secret: '',
      })
      setSuccess('Last.fm integration saved. Users can now connect Last.fm from Account.')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not save the Last.fm integration'))
    } finally {
      setSavingLastFMIntegration(false)
    }
  }

  const checkForUpdate = async () => {
    try {
      setUpdateAction('check')
      setError(null)
      const status = await adminUpdateAPI.check()
      setUpdateStatus(status)
      setSuccess(status.update_available ? `WaveNode ${status.latest_version} is available.` : 'WaveNode is up to date.')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not check for updates'))
    } finally {
      setUpdateAction(null)
    }
  }

  const runUpdate = async () => {
    if (!window.confirm('Update WaveNode now? The server may restart while the update is applied.')) {
      return
    }

    try {
      setUpdateAction('run')
      setError(null)
      const status = await adminUpdateAPI.run()
      setUpdateStatus(status)
      setSuccess('WaveNode update started. This page may disconnect while the server restarts.')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not start the update'))
    } finally {
      setUpdateAction(null)
    }
  }

  useEffect(() => {
    if (user?.role === 'admin') {
      void loadDashboard()
    }
  }, [loadDashboard, user?.role])

  useEffect(() => {
    if (!runningScan) return
    const interval = window.setInterval(() => void loadDashboard(true, false), 2000)
    return () => window.clearInterval(interval)
  }, [loadDashboard, runningScan])

  useEffect(() => {
    if (updateStatus?.state !== 'running' && updateStatus?.state !== 'checking') return
    const interval = window.setInterval(async () => {
      try {
        const status = await adminUpdateAPI.getStatus()
        setUpdateStatus(status)
        if (status?.state !== 'running' && status?.state !== 'checking') {
          const response = await api.get('/admin/system/status')
          setSystemStatus(response.data?.data || null)
        }
      } catch {
        // The server may briefly restart while applying an update.
      }
    }, 2500)
    return () => window.clearInterval(interval)
  }, [updateStatus?.state])

  useEffect(() => {
    if (user?.role !== 'admin') return

    return websocketService.onScanUpdate((update: WebSocketScanStatus) => {
      const scan = update as ScanStatus
      setScans(current => {
        const exists = current.some(item => item.id === scan.id)
        return exists
          ? current.map(item => item.id === scan.id ? scan : item)
          : [scan, ...current]
      })

      if (scan.status !== 'running' && scan.status !== 'pending' && scan.status !== 'stopping') {
        void loadDashboard(true)
      }
    })
  }, [loadDashboard, user?.role])

  const startJob = async (action: Exclude<JobAction, null>) => {
    const endpoints = {
      library: '/admin/scan/library',
      metadata: '/admin/enrich/musicbrainz',
      artwork: '/admin/enrich/cover-art',
    }
    const messages = {
      library: 'Library scan started',
      metadata: 'Metadata enrichment started',
      artwork: 'Cover art enrichment started',
    }

    try {
      setJobAction(action)
      setError(null)
      await api.post(endpoints[action])
      setSuccess(`${messages[action]}. Progress will update automatically.`)
      setActiveTab('library')
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, `Could not start ${action} job`))
    } finally {
      setJobAction(null)
    }
  }

  const stopLibraryScan = async () => {
    if (!window.confirm('Stop this library scan after the current file operation finishes?\n\nTracks already imported will be kept. Missing-track cleanup will not run from an incomplete discovery.')) return

    try {
      setStoppingLibraryScan(true)
      setError(null)
      await api.delete('/admin/scan/library')
      setSuccess('Stopping the library scan safely. The current file operation may take a moment to finish.')
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not stop the library scan'))
    } finally {
      setStoppingLibraryScan(false)
    }
  }

  const updateUserRole = async (target: User, role: User['role']) => {
    try {
      setError(null)
      await api.put(`/admin/users/${target.id}`, { role })
			setUsers(current => current.map(item => item.id === target.id ? {
				...item,
				role,
				...(role === 'admin' ? { library_restricted: false, music_source_ids: [] } : {}),
			} : item))
      setSuccess(`${target.username}'s role was updated`)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not update the user role'))
    }
  }

	const updateUserAccess = async (target: User, libraryRestricted: boolean, sourceIds: string[]) => {
		try {
			setUserAccessAction(target.id)
			setError(null)
			await api.put(`/admin/users/${target.id}`, {
				library_restricted: libraryRestricted,
				music_source_ids: sourceIds,
			})
			setUsers(current => current.map(item => item.id === target.id ? {
				...item,
				library_restricted: libraryRestricted,
				music_source_ids: sourceIds,
			} : item))
			setSuccess(`${target.username}'s library access was updated`)
		} catch (requestError) {
			setError(getErrorMessage(requestError, 'Could not update library access'))
		} finally {
			setUserAccessAction(null)
		}
	}

  const createUser = async (event: React.FormEvent) => {
    event.preventDefault()
    setCreatingUser(true)
    setError(null)
    try {
      const response = await api.post('/admin/users', {
        username: newUsername.trim(),
        email: newEmail.trim(),
        password: newPassword,
        role: newRole,
      })
      setUsers(current => [...current, response.data.data].sort((a, b) => a.username.localeCompare(b.username)))
      setNewUsername('')
      setNewEmail('')
      setNewPassword('')
      setNewRole('user')
      setShowUserForm(false)
      setSuccess('User account created')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not create the user'))
    } finally {
      setCreatingUser(false)
    }
  }

  const deleteUser = async (target: User) => {
    if (!window.confirm(`Delete ${target.username}? This cannot be undone.`)) return

    try {
      setError(null)
      await api.delete(`/admin/users/${target.id}`)
      setUsers(current => current.filter(item => item.id !== target.id))
      setSuccess(`${target.username} was deleted`)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not delete the user'))
    }
  }

  const clearHistory = async () => {
    if (!window.confirm('Clear all completed scan history?')) return

    try {
      await api.delete('/admin/scans')
      setScans(current => current.filter(scan => scan.status === 'running' || scan.status === 'pending'))
      setSuccess('Scan history was cleared')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not clear scan history'))
    }
  }

  const createMusicSource = async (path: string) => {
    try {
      setSourceAction('add')
      setError(null)
      const response = await api.post('/admin/library/sources', { path })
      setMusicSources(current => [...current, response.data?.data as MusicSource])
      setDirectoryPicker(null)
      setSuccess('Music source added. Run a library scan to import its tracks.')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not add the music source'))
    } finally {
      setSourceAction(null)
    }
  }

  const browseDirectories = async (path?: string) => {
    try {
      setDirectoryPickerLoading(true)
      setError(null)
      const response = await api.get('/admin/library/directories', {
        params: path ? { path } : undefined,
      })
      setDirectoryPicker(response.data?.data as DirectoryBrowserData)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not open that server folder'))
    } finally {
      setDirectoryPickerLoading(false)
    }
  }

  const removeMusicSource = async (source: MusicSource) => {
    if (!window.confirm(`Remove this music source?\n\n${source.path}\n\nExisting indexed tracks will remain until the library is cleared.`)) return

    try {
      setSourceAction(source.id)
      setError(null)
      await api.delete(`/admin/library/sources/${source.id}`)
      setMusicSources(current => current.filter(item => item.id !== source.id))
      setSuccess('Music source removed. Existing indexed tracks were not deleted.')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not remove the music source'))
    } finally {
      setSourceAction(null)
    }
  }

  const clearLibrary = async () => {
    if (!window.confirm('Clear the entire indexed library?\n\nThis removes all tracks and artists from WaveNode and empties playlist track lists. Your audio files and source paths will not be deleted.')) return

    try {
      setClearingLibrary(true)
      setError(null)
      await api.delete('/admin/library')
      setSuccess('The indexed music library was cleared. Your source folders were not changed.')
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not clear the music library'))
    } finally {
      setClearingLibrary(false)
    }
  }

  const saveAutomaticUpdates = async () => {
    try {
      setSavingAutomaticUpdates(true)
      setError(null)
      const response = await api.put('/admin/library/automatic-updates', automaticUpdates)
      setAutomaticUpdates(response.data?.data)
      setSuccess(automaticUpdates.enabled
        ? 'Automatic library updates enabled'
        : 'Automatic library updates disabled')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not save automatic update settings'))
    } finally {
      setSavingAutomaticUpdates(false)
    }
  }

  const downloadBackup = async () => {
    try {
      setBackupAction('download')
      setError(null)
      const response = await api.get('/admin/backup', { responseType: 'blob' })
      const disposition = String(response.headers['content-disposition'] || '')
      const filename = disposition.match(/filename="([^"]+)"/)?.[1] || 'wavenode-backup.zip'
      const url = URL.createObjectURL(response.data)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
      setSuccess('Backup downloaded')
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not create the backup'))
    } finally {
      setBackupAction(null)
    }
  }

  const restoreBackup = async (file?: File) => {
    if (!file) return
    if (!window.confirm('Restore this backup?\n\nCurrent users, playlists, library records, settings, and artwork will be replaced. Audio files are not changed.')) return

    try {
      setBackupAction('restore')
      setError(null)
      const formData = new FormData()
      formData.append('backup', file)
      await api.post('/admin/restore', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setSuccess('Backup restored. Reloading WaveNode...')
      window.setTimeout(() => {
        localStorage.removeItem('token')
        window.location.assign('/login')
      }, 1000)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not restore the backup'))
      setBackupAction(null)
    }
  }

  const discoverLocalArtistImages = async () => {
    try {
      setDiscoveringArtistImages(true)
      setError(null)
      await api.post('/admin/artists/discover-images')
      setSuccess('Local artist image discovery started. Progress will update automatically.')
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not search for local artist images'))
    } finally {
      setDiscoveringArtistImages(false)
    }
  }

  const uploadArtistImage = async (artist: AdminArtist, file?: File) => {
    if (!file) return

    try {
      setArtistImageAction(artist.id)
      setError(null)
      const formData = new FormData()
      formData.append('image', file)
      await api.post(`/admin/artists/${artist.id}/image`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
      setSuccess(`${artist.name}'s image was updated`)
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not upload the artist image'))
    } finally {
      setArtistImageAction(null)
    }
  }

  const removeArtistImage = async (artist: AdminArtist) => {
    if (!window.confirm(`Remove the image for ${artist.name}?`)) return

    try {
      setArtistImageAction(artist.id)
      setError(null)
      await api.delete(`/admin/artists/${artist.id}/image`)
      setSuccess(`${artist.name}'s image was removed`)
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not remove the artist image'))
    } finally {
      setArtistImageAction(null)
    }
  }

  const refreshArtistMetadata = async (artist: AdminArtist) => {
    try {
      setArtistImageAction(artist.id)
      setError(null)
      const result = await adminArtistImagesAPI.refreshMetadata(artist.id)
      const candidates = await adminArtistImagesAPI.listCandidates(artist.id)
      setArtistImageCandidates(current => ({ ...current, [artist.id]: candidates }))
      setCandidateArtistId(artist.id)
      setSuccess(result?.image ? `${artist.name}'s open-data image metadata was refreshed` : `No reusable open image was found for ${artist.name}`)
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not refresh artist metadata'))
    } finally {
      setArtistImageAction(null)
    }
  }

  const showArtistImageCandidates = async (artist: AdminArtist) => {
    if (candidateArtistId === artist.id) {
      setCandidateArtistId(null)
      return
    }
    try {
      setArtistImageAction(artist.id)
      setError(null)
      const candidates = await adminArtistImagesAPI.listCandidates(artist.id)
      setArtistImageCandidates(current => ({ ...current, [artist.id]: candidates }))
      setCandidateArtistId(artist.id)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not load artist image candidates'))
    } finally {
      setArtistImageAction(null)
    }
  }

  const selectArtistImageCandidate = async (artist: AdminArtist, image: ArtistImage) => {
    try {
      setArtistImageAction(artist.id)
      setError(null)
      await adminArtistImagesAPI.setPrimary(artist.id, image.id)
      const candidates = await adminArtistImagesAPI.listCandidates(artist.id)
      setArtistImageCandidates(current => ({ ...current, [artist.id]: candidates }))
      setSuccess(`${artist.name}'s preferred image was updated`)
      await loadDashboard(true)
    } catch (requestError) {
      setError(getErrorMessage(requestError, 'Could not select the artist image'))
    } finally {
      setArtistImageAction(null)
    }
  }

  const metadataPercent = stats?.enrichment.total_tracks
    ? Math.round((stats.enrichment.tracks_with_metadata / stats.enrichment.total_tracks) * 100)
    : 0
  const artworkPercent = stats?.enrichment.total_tracks
    ? Math.round((stats.enrichment.tracks_with_cover_art / stats.enrichment.total_tracks) * 100)
    : 0
  const artistImagePercent = stats?.enrichment.total_artists
    ? Math.round((stats.enrichment.artists_with_images / stats.enrichment.total_artists) * 100)
    : 0

  if (user?.role !== 'admin') {
    return (
      <Container>
        <AccessDenied>
          <div>
            <Shield size={58} color="#ff7777" />
            <h2>Administrator access required</h2>
            <Subtitle>This page controls server-wide operations and user permissions.</Subtitle>
          </div>
        </AccessDenied>
      </Container>
    )
  }

  return (
    <Container>
      <Header>
        <div>
          <Title>Server administration</Title>
          <Subtitle>Monitor your library, run maintenance jobs, and manage access.</Subtitle>
        </div>
        <LastUpdated>
          <Activity size={15} />
          {lastUpdated ? `Updated ${lastUpdated.toLocaleTimeString()}` : 'Loading server status'}
        </LastUpdated>
      </Header>

      {error && (
        <Notice $type="error">
          <span><AlertTriangle size={16} /> {error}</span>
          <DismissButton onClick={() => setError(null)} aria-label="Dismiss error"><X size={17} /></DismissButton>
        </Notice>
      )}
      {success && (
        <Notice $type="success">
          <span><CheckCircle2 size={16} /> {success}</span>
          <DismissButton onClick={() => setSuccess(null)} aria-label="Dismiss success message"><X size={17} /></DismissButton>
        </Notice>
      )}

      <Tabs aria-label="Administration sections">
        <TabButton $active={activeTab === 'overview'} onClick={() => setActiveTab('overview')}>
          <LayoutDashboard size={17} /> Overview
        </TabButton>
        <TabButton $active={activeTab === 'library'} onClick={() => setActiveTab('library')}>
          <ScanLine size={17} /> Library jobs
        </TabButton>
        <TabButton $active={activeTab === 'users'} onClick={() => setActiveTab('users')}>
          <Users size={17} /> Users
        </TabButton>
        <TabButton $active={activeTab === 'enrichment'} onClick={() => setActiveTab('enrichment')}>
          <Sparkles size={17} /> Enrichment
        </TabButton>
        <TabButton $active={activeTab === 'plugins'} onClick={() => setActiveTab('plugins')}>
          <Plug size={17} /> Plugins
        </TabButton>
        <TabButton $active={activeTab === 'integrations'} onClick={() => setActiveTab('integrations')}>
          <KeyRound size={17} /> Integrations
        </TabButton>
        <TabButton $active={activeTab === 'system'} onClick={() => setActiveTab('system')}>
          <Activity size={17} /> System
        </TabButton>
      </Tabs>

      {activeTab === 'overview' && (
        <>
          <StatsGrid>
            {[
              { icon: Music, value: stats?.total_tracks, label: 'Tracks' },
              { icon: Disc3, value: stats?.total_albums, label: 'Albums' },
              { icon: UserRound, value: stats?.enrichment.total_artists || stats?.total_artists, label: 'Artists' },
              { icon: ListMusic, value: stats?.total_playlists, label: 'Playlists' },
              { icon: Wifi, value: stats?.connected_users, label: 'Connected clients' },
            ].map(item => (
              <StatCard key={item.label}>
                <StatIcon><item.icon size={23} /></StatIcon>
                <StatValue>{loading ? '–' : formatNumber(item.value)}</StatValue>
                <StatLabel>{item.label}</StatLabel>
              </StatCard>
            ))}
          </StatsGrid>

          <TwoColumn>
            <Panel>
              <PanelHeader>
                <div>
                  <PanelTitle>Library readiness</PanelTitle>
                  <PanelDescription>Coverage of metadata and artwork across the collection.</PanelDescription>
                </div>
                <Button onClick={() => void loadDashboard(true)} disabled={refreshing}>
                  <Spin size={15} $active={refreshing} /> Refresh
                </Button>
              </PanelHeader>
              <PanelBody>
                <CoverageRow>
                  <CoverageHeader><span>Track metadata</span><strong>{metadataPercent}%</strong></CoverageHeader>
                  <ProgressTrack><ProgressFill $progress={metadataPercent} /></ProgressTrack>
                </CoverageRow>
                <CoverageRow>
                  <CoverageHeader><span>Cover artwork</span><strong>{artworkPercent}%</strong></CoverageHeader>
                  <ProgressTrack><ProgressFill $progress={artworkPercent} /></ProgressTrack>
                </CoverageRow>
                <CoverageRow>
                  <CoverageHeader><span>Artist images</span><strong>{artistImagePercent}%</strong></CoverageHeader>
                  <ProgressTrack><ProgressFill $progress={artistImagePercent} /></ProgressTrack>
                </CoverageRow>
              </PanelBody>
            </Panel>

            <Panel>
              <PanelHeader>
                <div>
                  <PanelTitle>Current activity</PanelTitle>
                  <PanelDescription>Long-running jobs update automatically.</PanelDescription>
                </div>
              </PanelHeader>
              <PanelBody>
                {runningScan ? (
                  <CurrentJob>
                    <JobHeader>
                      <PrimaryText>{scanLabel(runningScan.type)}</PrimaryText>
                      <StatusBadge $status={runningScan.status}>{statusIcon(runningScan.status)} {runningScan.status}</StatusBadge>
                    </JobHeader>
                    <ProgressTrack><ProgressFill $progress={getScanProgress(runningScan)} /></ProgressTrack>
                    <SecondaryText>{runningScan.processed} of {runningScan.total_files} items · {getScanProgress(runningScan)}%</SecondaryText>
                    {runningScan.current_file && <JobFile title={runningScan.current_file}>{runningScan.current_file}</JobFile>}
                  </CurrentJob>
                ) : (
                  <EmptyState>
                    <EmptyIcon><CheckCircle2 size={34} /></EmptyIcon>
                    No maintenance jobs are running.
                  </EmptyState>
                )}
              </PanelBody>
            </Panel>
          </TwoColumn>
        </>
      )}

      {activeTab === 'library' && (
        <LibraryStack>
          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Music sources</PanelTitle>
                <PanelDescription>Choose the server folders WaveNode scans for music.</PanelDescription>
              </div>
              <Button $variant="primary" onClick={() => void browseDirectories()} disabled={Boolean(runningScan) || directoryPickerLoading}>
                <FolderOpen size={16} /> {directoryPickerLoading ? 'Opening...' : 'Add source'}
              </Button>
            </PanelHeader>
            <PanelBody>
              <SourceHelp>
                Select a folder visible inside the WaveNode server or Docker container.
                Host folders must be mounted into Docker before they appear in the folder picker.
              </SourceHelp>

              {musicSources.length ? (
                <SourceList>
                  {musicSources.map(source => (
                    <SourceRow key={source.id}>
                      <SourcePath title={source.path}>
                        <FolderOpen size={18} color={theme.colors.accent} />
                        <span>{source.path}</span>
                      </SourcePath>
                      <Button
                        $variant="danger"
                        onClick={() => void removeMusicSource(source)}
                        disabled={Boolean(runningScan) || sourceAction !== null}
                      >
                        <Trash2 size={15} /> {sourceAction === source.id ? 'Removing...' : 'Remove'}
                      </Button>
                    </SourceRow>
                  ))}
                </SourceList>
              ) : (
                <EmptyState>
                  <EmptyIcon><FolderOpen size={34} /></EmptyIcon>
                  No music sources configured. Add a server-visible folder before scanning.
                </EmptyState>
              )}

              <DangerZone>
                <div>
                  <PrimaryText>Clear indexed library</PrimaryText>
                  <SecondaryText>Removes tracks and artists from WaveNode without deleting source folders or audio files.</SecondaryText>
                </div>
                <Button $variant="danger" onClick={() => void clearLibrary()} disabled={Boolean(runningScan) || clearingLibrary || !stats?.total_tracks}>
                  <Trash2 size={15} /> {clearingLibrary ? 'Clearing...' : 'Clear entire library'}
                </Button>
              </DangerZone>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Backup and restore</PanelTitle>
                <PanelDescription>Protect accounts, playlists, indexed library data, settings, and stored artwork.</PanelDescription>
              </div>
              <ButtonRow>
                <Button onClick={() => void downloadBackup()} disabled={backupAction !== null}>
                  <HardDrive size={16} /> {backupAction === 'download' ? 'Preparing...' : 'Download backup'}
                </Button>
                <Button as="label" $variant="danger">
                  <Upload size={16} /> {backupAction === 'restore' ? 'Restoring...' : 'Restore backup'}
                  <input
                    type="file"
                    accept=".zip,application/zip"
                    hidden
                    disabled={backupAction !== null}
                    onChange={event => {
                      void restoreBackup(event.target.files?.[0])
                      event.currentTarget.value = ''
                    }}
                  />
                </Button>
              </ButtonRow>
            </PanelHeader>
            <PanelBody>
              <SourceHelp>
                Backups do not contain your original audio files. Keep the configured music folders backed up separately.
              </SourceHelp>
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Automatic library updates</PanelTitle>
                <PanelDescription>Detect changed, added, or removed audio files and run scheduled safety scans.</PanelDescription>
              </div>
            </PanelHeader>
            <PanelBody>
              <AutomationRow>
                <div>
                  <label>
                    <input
                      type="checkbox"
                      checked={automaticUpdates.enabled}
                      onChange={event => setAutomaticUpdates(current => ({
                        ...current,
                        enabled: event.target.checked,
                      }))}
                    />
                    Enable automatic updates
                  </label>
                  <SecondaryText>WaveNode checks source folders every 30 seconds.</SecondaryText>
                </div>
                <label>
                  Safety scan interval
                  <input
                    type="number"
                    min={1}
                    max={10080}
                    value={automaticUpdates.interval_minutes}
                    onChange={event => setAutomaticUpdates(current => ({
                      ...current,
                      interval_minutes: Number(event.target.value),
                    }))}
                  />
                  <span>minutes</span>
                </label>
                <Button
                  $variant="primary"
                  onClick={() => void saveAutomaticUpdates()}
                  disabled={savingAutomaticUpdates}
                >
                  <RefreshCw size={15} /> {savingAutomaticUpdates ? 'Saving...' : 'Save'}
                </Button>
              </AutomationRow>
              {(automaticUpdates.last_reason || automaticUpdates.last_checked_at || automaticUpdates.last_error) && (
                <SourceHelp>
                  {automaticUpdates.last_error
                    ? `Last check failed: ${automaticUpdates.last_error}`
                    : `${automaticUpdates.last_reason || 'Monitoring active'}${automaticUpdates.last_checked_at ? ` · checked ${formatDate(automaticUpdates.last_checked_at)}` : ''}`}
                </SourceHelp>
              )}
            </PanelBody>
          </Panel>

          <Panel>
          <PanelHeader>
            <div>
              <PanelTitle>Library jobs</PanelTitle>
              <PanelDescription>Scan all configured music sources and review previous runs.</PanelDescription>
            </div>
            <ButtonRow>
              <Button $variant="primary" onClick={() => void startJob('library')} disabled={Boolean(runningScan) || jobAction !== null || musicSources.length === 0}>
                <Database size={16} /> {jobAction === 'library' ? 'Starting…' : 'Scan library'}
              </Button>
              {runningScan?.type === 'library' && (
                <Button
                  $variant="danger"
                  onClick={() => void stopLibraryScan()}
                  disabled={runningScan.status === 'stopping' || stoppingLibraryScan}
                >
                  <Square size={14} fill="currentColor" />
                  {runningScan.status === 'stopping' || stoppingLibraryScan ? 'Stopping...' : 'Stop scan'}
                </Button>
              )}
              <Button onClick={() => void loadDashboard(true)} disabled={refreshing}>
                <Spin size={15} $active={refreshing} /> Refresh
              </Button>
              <Button $variant="danger" onClick={clearHistory} disabled={completedScans.length === 0}>
                <Trash2 size={15} /> Clear history
              </Button>
            </ButtonRow>
          </PanelHeader>
          <PanelBody>
            {runningScan && (
              <CurrentJob>
                <JobHeader>
                  <PrimaryText>{scanLabel(runningScan.type)}</PrimaryText>
                  <StatusBadge $status={runningScan.status}>{statusIcon(runningScan.status)} {runningScan.status}</StatusBadge>
                </JobHeader>
                <ProgressTrack><ProgressFill $progress={getScanProgress(runningScan)} /></ProgressTrack>
                <SecondaryText>{runningScan.processed} / {runningScan.total_files} processed · {getScanProgress(runningScan)}%</SecondaryText>
                {runningScan.current_file && <JobFile title={runningScan.current_file}>{runningScan.current_file}</JobFile>}
              </CurrentJob>
            )}

            <h3 style={{ margin: runningScan ? '22px 0 12px' : '0 0 12px' }}>Recent jobs</h3>
            {completedScans.length ? (
              <HistoryList>
                {completedScans.map(scan => (
                  <HistoryItem key={scan.id}>
                    <div>
                      <PrimaryText>{scanLabel(scan.type)}</PrimaryText>
                      <SecondaryText>{formatDate(scan.started_at)}</SecondaryText>
                    </div>
                    <div>
                      <PrimaryText>{formatNumber(scan.processed)} items</PrimaryText>
                      <SecondaryText>{scanResultText(scan)}</SecondaryText>
                    </div>
                    <div>
                      <PrimaryText>{getScanProgress(scan)}% complete</PrimaryText>
                      <SecondaryText>{scan.tracks_skipped || 0} skipped</SecondaryText>
                    </div>
                    <StatusBadge $status={scan.status}>{statusIcon(scan.status)} {scan.status}</StatusBadge>
                  </HistoryItem>
                ))}
              </HistoryList>
            ) : (
              <EmptyState><EmptyIcon><Database size={34} /></EmptyIcon>No completed jobs yet.</EmptyState>
            )}
          </PanelBody>
        </Panel>
        </LibraryStack>
      )}

      {activeTab === 'users' && (
        <Panel>
          <PanelHeader>
            <div>
              <PanelTitle>User access</PanelTitle>
              <PanelDescription>Create accounts, change roles, or remove access. The final administrator is protected.</PanelDescription>
            </div>
            <ButtonRow>
              <Button $variant="primary" onClick={() => setShowUserForm(current => !current)}>
                <UserPlus size={15} /> Add user
              </Button>
              <Button onClick={() => void loadDashboard(true)} disabled={refreshing}>
                <Spin size={15} $active={refreshing} /> Refresh
              </Button>
            </ButtonRow>
          </PanelHeader>
          {showUserForm && (
            <UserForm onSubmit={event => void createUser(event)}>
              <input aria-label="New username" placeholder="Username" minLength={3} required value={newUsername} onChange={event => setNewUsername(event.target.value)} />
              <input aria-label="New user email" placeholder="Email" type="email" required value={newEmail} onChange={event => setNewEmail(event.target.value)} />
              <input aria-label="New user password" placeholder="Password (8+ characters)" type="password" minLength={8} required value={newPassword} onChange={event => setNewPassword(event.target.value)} />
              <select aria-label="New user role" value={newRole} onChange={event => setNewRole(event.target.value as User['role'])}>
                <option value="user">User</option>
                <option value="admin">Administrator</option>
              </select>
              <Button $variant="primary" type="submit" disabled={creatingUser}>
                {creatingUser ? 'Creating...' : 'Create account'}
              </Button>
            </UserForm>
          )}
          {users.length ? (
            <TableWrap>
              <Table>
                <thead>
					<tr><th>User</th><th>Email</th><th>Role</th><th>Music access</th><th>Joined</th><th>Action</th></tr>
                </thead>
                <tbody>
                  {users.map(target => {
                    const isCurrentUser = target.id === user.id
                    const protectedAdmin = target.role === 'admin' && users.filter(item => item.role === 'admin').length === 1
                    return (
                      <tr key={target.id}>
                        <td><PrimaryText>{target.username}</PrimaryText>{isCurrentUser && <SecondaryText>Current account</SecondaryText>}</td>
                        <td>{target.email}</td>
                        <td>
                          <Select
                            value={target.role}
                            disabled={isCurrentUser || protectedAdmin}
                            onChange={event => void updateUserRole(target, event.target.value as User['role'])}
                          >
                            <option value="user">User</option>
                            <option value="admin">Administrator</option>
                          </Select>
                        </td>
						<td>
							{target.role === 'admin' ? (
								<SecondaryText>All folders (administrator)</SecondaryText>
							) : (
								<AccessControl>
									<label>
										<input
											type="checkbox"
											checked={target.library_restricted || false}
											disabled={userAccessAction === target.id}
											onChange={event => void updateUserAccess(target, event.target.checked, event.target.checked ? (target.music_source_ids || []) : [])}
										/>
										Limit to selected folders
									</label>
									{target.library_restricted && (
										<SourceChoices>
											{musicSources.length ? musicSources.map(source => {
												const selected = (target.music_source_ids || []).includes(source.id)
												return (
													<label key={source.id} title={source.path}>
														<input
															type="checkbox"
															checked={selected}
															disabled={userAccessAction === target.id}
															onChange={() => void updateUserAccess(target, true, selected
																? target.music_source_ids.filter(id => id !== source.id)
																: [...(target.music_source_ids || []), source.id])}
														/>
														{source.path}
													</label>
												)
											}) : <SecondaryText>No music sources configured</SecondaryText>}
											{musicSources.length > 0 && (target.music_source_ids || []).length === 0 && (
												<SecondaryText>No folders selected: music is fully blocked.</SecondaryText>
											)}
										</SourceChoices>
									)}
								</AccessControl>
							)}
						</td>
                        <td>{formatDate(target.created_at)}</td>
                        <td>
                          <Button
                            $variant="danger"
                            disabled={isCurrentUser || protectedAdmin}
                            onClick={() => void deleteUser(target)}
                          >
                            <Trash2 size={14} /> Delete
                          </Button>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </Table>
            </TableWrap>
          ) : (
            <EmptyState><EmptyIcon><Users size={34} /></EmptyIcon>No users found.</EmptyState>
          )}
        </Panel>
      )}

      {activeTab === 'enrichment' && (
        <LibraryStack>
          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Library enrichment</PanelTitle>
                <PanelDescription>Fill missing metadata and artwork.</PanelDescription>
              </div>
            </PanelHeader>
            <PanelBody>
              <ActionGrid>
                <ActionCard>
                  <ActionIcon><Tags size={21} /></ActionIcon>
                  <ActionTitle>Artist metadata</ActionTitle>
                  <ActionText>Look up artist information for library entries that do not yet have enriched metadata.</ActionText>
                  <Button $variant="primary" onClick={() => void startJob('metadata')} disabled={Boolean(runningScan) || jobAction !== null}>
                    <Tags size={15} /> {jobAction === 'metadata' ? 'Starting…' : 'Enrich metadata'}
                  </Button>
                </ActionCard>
                <ActionCard>
                  <ActionIcon><Image size={21} /></ActionIcon>
                  <ActionTitle>Cover artwork</ActionTitle>
                  <ActionText>Find album artwork for tracks that currently use generated fallback covers.</ActionText>
                  <Button $variant="primary" onClick={() => void startJob('artwork')} disabled={Boolean(runningScan) || jobAction !== null}>
                    <Image size={15} /> {jobAction === 'artwork' ? 'Starting…' : 'Find cover artwork'}
                  </Button>
                </ActionCard>
                <ActionCard>
                  <ActionIcon><UserRound size={21} /></ActionIcon>
                  <ActionTitle>Local artist images</ActionTitle>
                  <ActionText>Search artist folders first, then use embedded artwork from audio files. No external service is used.</ActionText>
                  <Button $variant="primary" onClick={() => void discoverLocalArtistImages()} disabled={Boolean(runningScan) || discoveringArtistImages || artistImageAction !== null}>
                    <ScanLine size={15} /> {discoveringArtistImages ? 'Searching…' : 'Find local images'}
                  </Button>
                </ActionCard>
              </ActionGrid>
              {artistImageScan && (
                <CurrentJob style={{ marginTop: 18 }}>
                  <JobHeader>
                    <PrimaryText>{scanLabel(artistImageScan.type)}</PrimaryText>
                    <StatusBadge $status={artistImageScan.status}>
                      {statusIcon(artistImageScan.status)} {artistImageScan.status.replace(/_/g, ' ')}
                    </StatusBadge>
                  </JobHeader>
                  <ProgressTrack><ProgressFill $progress={getScanProgress(artistImageScan)} /></ProgressTrack>
                  <SecondaryText>
                    {artistImageScan.processed} of {artistImageScan.total_files} artists · {getScanProgress(artistImageScan)}%
                  </SecondaryText>
                  <SecondaryText>{scanResultText(artistImageScan)}</SecondaryText>
                  {artistImageScan.current_file && (
                    <JobFile title={artistImageScan.current_file}>Checking {artistImageScan.current_file}</JobFile>
                  )}
                  {artistImageScan.errors && artistImageScan.errors.length > 0 && (
                    <ErrorSummary>
                      <strong>{artistImageScan.errors.length} issue{artistImageScan.errors.length === 1 ? '' : 's'}</strong>
                      {artistImageScan.errors.slice(-5).map((message, index) => (
                        <span key={`${message}-${index}`}>{message}</span>
                      ))}
                    </ErrorSummary>
                  )}
                </CurrentJob>
              )}
              {runningScan && (
                <Notice $type="error" style={{ marginTop: 18, marginBottom: 0 }}>
                  <span><Clock3 size={16} /> Wait for the current {scanLabel(runningScan.type)} to finish before starting another job.</span>
                </Notice>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Artist images</PanelTitle>
                <PanelDescription>Refresh open-data images, view licenses, choose a preferred image, or upload a permitted fallback.</PanelDescription>
              </div>
              <Button onClick={() => void loadDashboard(true)} disabled={refreshing}>
                <Spin size={15} $active={refreshing} /> Refresh
              </Button>
            </PanelHeader>
            <PanelBody>
              {artists.length ? (
                <ArtistImageList>
                  {artists.map(artist => {
                    const imageUrl = resolveMediaUrl(
                      artist.image_medium_url || artist.image_url || artist.image_small_url || artist.image_large_url,
                    )
                    const inputId = `artist-image-${artist.id}`
                    return (
                      <ArtistImageRow key={artist.id}>
                        <ArtistImagePreview $imageUrl={imageUrl} $fallback={getArtworkGradient(artist.name)}>
                          {imageUrl ? null : <UserRound size={24} />}
                        </ArtistImagePreview>
                        <div>
                          <PrimaryText>{artist.name}</PrimaryText>
                          <SecondaryText>{artist.track_count} tracks · {imageUrl ? 'Image set' : 'Generated fallback'}</SecondaryText>
                        </div>
                        <ButtonRow>
                          <HiddenFileInput
                            id={inputId}
                            type="file"
                            accept="image/jpeg,image/png,image/webp,image/gif"
                            onChange={event => {
                              void uploadArtistImage(artist, event.target.files?.[0])
                              event.target.value = ''
                            }}
                          />
                          <Button as="label" htmlFor={inputId} $variant="primary">
                            <Upload size={14} /> {artistImageAction === artist.id ? 'Uploading…' : imageUrl ? 'Replace' : 'Upload'}
                          </Button>
                          <Button onClick={() => void refreshArtistMetadata(artist)} disabled={artistImageAction !== null}>
                            <RefreshCw size={14} /> Refresh
                          </Button>
                          <Button onClick={() => void showArtistImageCandidates(artist)} disabled={artistImageAction !== null}>
                            <Image size={14} /> Candidates
                          </Button>
                          {imageUrl && (
                            <Button $variant="danger" onClick={() => void removeArtistImage(artist)} disabled={artistImageAction !== null}>
                              <Trash2 size={14} /> Remove
                            </Button>
                          )}
                        </ButtonRow>
                        {candidateArtistId === artist.id && (
                          <CandidateList>
                            {(artistImageCandidates[artist.id] || []).length ? (
                              (artistImageCandidates[artist.id] || []).map(candidate => (
                                <CandidateRow key={candidate.id}>
                                  <CandidateImage src={resolveMediaUrl(candidate.thumbnail_url || candidate.image_url)} alt="" />
                                  <div>
                                    <PrimaryText>{candidate.source.replace(/_/g, ' ')} {candidate.is_primary ? 'Primary' : ''}</PrimaryText>
                                    <SecondaryText>
                                      {candidate.license_name || 'License not provided'} {candidate.author_name ? ` by ${candidate.author_name}` : ''}
                                    </SecondaryText>
                                    {candidate.attribution_text && <SecondaryText>{candidate.attribution_text}</SecondaryText>}
                                  </div>
                                  <ButtonRow>
                                    {candidate.source_page_url && (
                                      <Button as="a" href={candidate.source_page_url} target="_blank" rel="noreferrer">
                                        Source
                                      </Button>
                                    )}
                                    <Button
                                      $variant="primary"
                                      onClick={() => void selectArtistImageCandidate(artist, candidate)}
                                      disabled={candidate.is_primary || artistImageAction !== null}
                                    >
                                      Select
                                    </Button>
                                  </ButtonRow>
                                </CandidateRow>
                              ))
                            ) : (
                              <SecondaryText>No reusable candidates saved yet. Refresh metadata to search MusicBrainz, Wikidata, and Commons.</SecondaryText>
                            )}
                          </CandidateList>
                        )}
                      </ArtistImageRow>
                    )
                  })}
                </ArtistImageList>
              ) : (
                <EmptyState><EmptyIcon><UserRound size={34} /></EmptyIcon>No artists found. Scan the library first.</EmptyState>
              )}
            </PanelBody>
          </Panel>
        </LibraryStack>
      )}

      {activeTab === 'plugins' && (
        <Panel>
          <PanelHeader>
            <div>
              <PanelTitle>Plugin manager</PanelTitle>
              <PanelDescription>
                Install declarative extensions that add approved content to WaveNode. Plugins cannot execute arbitrary code.
              </PanelDescription>
            </div>
            <ButtonRow>
              <Button $variant="primary" onClick={() => setShowPluginForm(current => !current)}>
                <PackagePlus size={15} /> Install plugin
              </Button>
              <Button onClick={() => void loadDashboard(true)} disabled={refreshing}>
                <Spin size={15} $active={refreshing} /> Refresh
              </Button>
            </ButtonRow>
          </PanelHeader>
          {showPluginForm && (
            <PluginForm onSubmit={event => void installPlugin(event)}>
              <div>
                <PrimaryText>Plugin manifest</PrimaryText>
                <SecondaryText>
                  Paste a schema version 1 manifest. This starter adds a track download action and can be edited before installation.
                </SecondaryText>
              </div>
              <textarea
                aria-label="Plugin manifest JSON"
                spellCheck={false}
                value={pluginManifest}
                onChange={event => setPluginManifest(event.target.value)}
              />
              <ButtonRow>
                <Button $variant="primary" type="submit" disabled={pluginAction !== null}>
                  <PackagePlus size={15} /> {pluginAction === 'install' ? 'Installing...' : 'Validate and install'}
                </Button>
                <Button type="button" onClick={() => setPluginManifest(downloadPluginTemplate)}>
                  Restore download example
                </Button>
                <Button type="button" onClick={() => setShowPluginForm(false)}>Cancel</Button>
              </ButtonRow>
            </PluginForm>
          )}
          <PanelBody>
            {plugins.length ? (
              <SourceList>
                {plugins.map(plugin => {
                  const manifest = plugin.manifest as {
                    description?: string
                    contributes?: { home_rows?: unknown[] }
                  }
                  const rowCount = manifest.contributes?.home_rows?.length || 0
                  return (
                    <SourceRow key={plugin.id}>
                      <div>
                        <PrimaryText>{plugin.name}</PrimaryText>
                        <SecondaryText>{manifest.description || plugin.id}</SecondaryText>
                        <PluginMeta>
                          <span>v{plugin.version}</span>
                          <span>{plugin.id}</span>
                          <span>{rowCount} home row{rowCount === 1 ? '' : 's'}</span>
                          <span>Updated {formatDate(plugin.updated_at)}</span>
                        </PluginMeta>
                      </div>
                      <ButtonRow>
                        <Button
                          disabled={pluginAction !== null}
                          onClick={() => editPlugin(plugin)}
                        >
                          Edit manifest
                        </Button>
                        <Button
                          $variant={plugin.enabled ? 'secondary' : 'primary'}
                          disabled={pluginAction !== null}
                          onClick={() => void setPluginEnabled(plugin, !plugin.enabled)}
                        >
                          <Power size={14} /> {plugin.enabled ? 'Disable' : 'Enable'}
                        </Button>
                        <Button
                          $variant="danger"
                          disabled={pluginAction !== null}
                          onClick={() => void removePlugin(plugin)}
                        >
                          <Trash2 size={14} /> Remove
                        </Button>
                      </ButtonRow>
                    </SourceRow>
                  )
                })}
              </SourceList>
            ) : (
              <EmptyState>
                <EmptyIcon><Plug size={34} /></EmptyIcon>
                No plugins installed. Use the included download example to add the first track menu action.
              </EmptyState>
            )}
          </PanelBody>
        </Panel>
      )}

      {activeTab === 'integrations' && (
        <Panel>
          <PanelHeader>
            <div>
              <PanelTitle>External services</PanelTitle>
              <PanelDescription>
                Configure server-wide credentials once. Users only connect their own account from the Account page.
              </PanelDescription>
            </div>
            <Button onClick={() => void loadDashboard(true)} disabled={refreshing}>
              <Spin size={15} $active={refreshing} /> Refresh
            </Button>
          </PanelHeader>
          <PanelBody>
            <IntegrationForm>
              <div>
                <PrimaryText>Last.fm application</PrimaryText>
                <SecondaryText>
                  Save the Last.fm API key and shared secret for this WaveNode server. The shared secret is hidden after saving.
                </SecondaryText>
              </div>
              <label>
                API key
                <input
                  value={lastFMIntegration.api_key}
                  onChange={event => setLastFMIntegration(current => ({ ...current, api_key: event.target.value }))}
                  placeholder="Last.fm API key"
                />
              </label>
              <label>
                Shared secret
                <input
                  type="password"
                  value={lastFMIntegration.shared_secret || ''}
                  onChange={event => setLastFMIntegration(current => ({ ...current, shared_secret: event.target.value }))}
                  placeholder={lastFMIntegration.has_shared_secret ? 'Saved. Leave blank to keep existing secret.' : 'Last.fm shared secret'}
                />
              </label>
              <ButtonRow>
                <Button $variant="primary" onClick={() => void saveLastFMIntegration()} disabled={savingLastFMIntegration}>
                  <KeyRound size={15} /> {savingLastFMIntegration ? 'Saving...' : 'Save Last.fm integration'}
                </Button>
                <StatusBadge $status={lastFMIntegration.configured ? 'completed' : 'failed'}>
                  {lastFMIntegration.configured ? <CheckCircle2 size={14} /> : <AlertTriangle size={14} />}
                  {lastFMIntegration.configured ? 'Configured' : 'Not configured'}
                </StatusBadge>
              </ButtonRow>
            </IntegrationForm>
          </PanelBody>
        </Panel>
      )}

      {activeTab === 'system' && (
        <LibraryStack>
          <StatsGrid>
            {[
              { icon: Shield, value: systemStatus ? formatVersion(systemStatus.version) : null, label: 'WaveNode' },
              { icon: Clock3, value: systemStatus ? formatDuration(systemStatus.uptime_seconds) : null, label: 'Uptime' },
              { icon: Wifi, value: systemStatus?.active_streams, label: 'Active streams' },
              { icon: Database, value: systemStatus?.database_in_use, label: 'Database connections' },
              { icon: Image, value: systemStatus ? formatBytes(systemStatus.artwork_bytes) : null, label: 'Artwork storage' },
            ].map(item => (
              <StatCard key={item.label}>
                <StatIcon><item.icon size={23} /></StatIcon>
                <StatValue>{loading ? '-' : item.value ?? '-'}</StatValue>
                <StatLabel>{item.label}</StatLabel>
              </StatCard>
            ))}
          </StatsGrid>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>WaveNode updates</PanelTitle>
                <PanelDescription>Check GitHub releases and apply server updates from this dashboard.</PanelDescription>
              </div>
              <ButtonRow>
                <Button onClick={() => void checkForUpdate()} disabled={updateAction !== null || updateStatus?.state === 'running' || updateStatus?.state === 'checking'}>
                  <RefreshCw size={15} /> {updateAction === 'check' ? 'Checking...' : 'Check for updates'}
                </Button>
                <Button
                  $variant="primary"
                  onClick={() => void runUpdate()}
                  disabled={
                    updateAction !== null ||
                    updateStatus?.state === 'running' ||
                    updateStatus?.state === 'checking' ||
                    !updateStatus?.command_configured ||
                    !updateStatus?.update_available
                  }
                >
                  <RefreshCw size={15} /> {updateStatus?.state === 'running' || updateAction === 'run' ? 'Updating...' : 'Update now'}
                </Button>
              </ButtonRow>
            </PanelHeader>
            <PanelBody>
              <SourceList>
                <SourceRow>
                  <span>Installed version</span>
                  <PrimaryText>{formatVersion(updateStatus?.current_version || systemStatus?.version)}</PrimaryText>
                </SourceRow>
                <SourceRow>
                  <span>Latest release</span>
                  <PrimaryText>
                    {updateStatus?.latest_version ? formatVersion(updateStatus.latest_version) : 'Not checked yet'}
                    {updateStatus?.release_url && (
                      <>
                        {' '}
                        <a href={updateStatus.release_url} target="_blank" rel="noreferrer">View release</a>
                      </>
                    )}
                  </PrimaryText>
                </SourceRow>
                <SourceRow>
                  <span>Update status</span>
                  <StatusBadge $status={
                    updateStatus?.state === 'failed' || updateStatus?.state === 'unavailable'
                      ? 'failed'
                      : updateStatus?.state === 'running' || updateStatus?.state === 'checking'
                        ? 'running'
                        : 'completed'
                  }>
                    {updateStatus?.state === 'failed' || updateStatus?.state === 'unavailable'
                      ? <AlertTriangle size={14} />
                      : updateStatus?.state === 'running' || updateStatus?.state === 'checking'
                        ? <Clock3 size={14} />
                        : <CheckCircle2 size={14} />}
                    {updateStatus?.message || 'Update status unavailable'}
                  </StatusBadge>
                </SourceRow>
                <SourceRow>
                  <span>One-click update</span>
                  <PrimaryText>{updateStatus?.command_configured ? 'Configured' : 'Not configured'}</PrimaryText>
                </SourceRow>
              </SourceList>
              {!updateStatus?.command_configured && (
                <SourceHelp style={{ marginTop: 14, marginBottom: 0 }}>
                  Set <code>WAVENODE_UPDATE_COMMAND</code> on the server to enable the Update now button.
                  The command runs server-side and should pull the latest release, rebuild or pull containers, and restart WaveNode.
                </SourceHelp>
              )}
              {updateStatus?.log_tail?.length ? (
                <LogPanel aria-label="Latest update log">
                  {updateStatus.log_tail.map((line, index) => (
                    <div key={`${index}-${line}`}>{line}</div>
                  ))}
                </LogPanel>
              ) : null}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Library health</PanelTitle>
                <PanelDescription>Audio files, metadata, artwork, sources, and storage checked in one place.</PanelDescription>
              </div>
              <Button onClick={() => void loadDashboard(true)} disabled={refreshing}>
                <Spin size={15} $active={refreshing} /> Check again
              </Button>
            </PanelHeader>
            <PanelBody>
              <HealthOverview>
                <HealthScoreCard $score={diagnostics?.health_score ?? 100}>
                  <HealthScoreValue>{loading ? '-' : `${diagnostics?.health_score ?? 100}%`}</HealthScoreValue>
                  <HealthScoreLabel>
                    {(diagnostics?.health_score ?? 100) >= 90
                      ? 'Library is healthy'
                      : (diagnostics?.health_score ?? 100) >= 70
                        ? 'Library needs attention'
                        : 'Library has serious issues'}
                  </HealthScoreLabel>
                  <SecondaryText>
                    {formatNumber(diagnostics?.healthy_tracks)} of {formatNumber(diagnostics?.indexed_tracks)} tracks passed all checks
                  </SecondaryText>
                </HealthScoreCard>
                <DiagnosticGrid>
                  {[
                    { label: 'Missing audio files', value: diagnostics?.missing_files || 0 },
                    { label: 'Duplicate groups', value: diagnostics?.duplicate_groups || 0 },
                    { label: 'Metadata issues', value: diagnostics?.invalid_metadata || 0 },
                    { label: 'Unsupported formats', value: diagnostics?.unsupported_formats || 0 },
                    { label: 'Tracks without artwork', value: diagnostics?.missing_artwork || 0 },
                    { label: 'Source or storage alerts', value: (diagnostics?.unavailable_sources || 0) + (diagnostics?.low_space_sources || 0) },
                  ].map(item => (
                    <DiagnosticCard key={item.label} $problem={item.value > 0}>
                      <DiagnosticValue>{formatNumber(item.value)}</DiagnosticValue>
                      <DiagnosticLabel>{item.label}</DiagnosticLabel>
                    </DiagnosticCard>
                  ))}
                </DiagnosticGrid>
              </HealthOverview>
              {diagnostics?.details_truncated && (
                <SourceHelp style={{ marginTop: 14, marginBottom: 0 }}>
                  Showing the first 100 examples in each category. The totals above include the entire library.
                </SourceHelp>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Music sources and storage</PanelTitle>
                <PanelDescription>Folder access and remaining space on every configured music source.</PanelDescription>
              </div>
            </PanelHeader>
            <PanelBody>
              {diagnostics?.sources.length ? (
                <SourceList>
                  {diagnostics.sources.map(source => (
                    <SourceRow key={source.path}>
                      <FolderOpen size={20} color={source.accessible ? theme.colors.accent : '#ff7777'} />
                      <StorageDetails>
                        <StorageHeader>
                          <SourcePath title={source.path}><span>{source.path}</span></SourcePath>
                          <StatusBadge
                            $status={source.space_status === 'healthy' ? 'completed' : source.space_status === 'unavailable' || source.space_status === 'critical' ? 'failed' : 'running'}
                            title={source.error}
                          >
                            {source.space_status === 'healthy' ? <CheckCircle2 size={14} /> : <AlertTriangle size={14} />}
                            {source.space_status === 'healthy'
                              ? 'Healthy'
                              : source.space_status === 'unavailable'
                                ? 'Unavailable'
                                : source.space_status === 'unknown'
                                  ? 'Space unknown'
                                  : `${source.space_status} space`}
                          </StatusBadge>
                        </StorageHeader>
                        {source.accessible && source.total_bytes > 0 ? (
                          <>
                            <StorageProgress>
                              <StorageProgressFill $percent={source.used_percent} $status={source.space_status} />
                            </StorageProgress>
                            <SecondaryText>
                              {formatBytes(source.free_bytes)} free of {formatBytes(source.total_bytes)} · {source.used_percent.toFixed(1)}% used
                            </SecondaryText>
                          </>
                        ) : (
                          <SecondaryText>{source.error || 'Storage capacity is not available.'}</SecondaryText>
                        )}
                      </StorageDetails>
                    </SourceRow>
                  ))}
                </SourceList>
              ) : (
                <EmptyState><EmptyIcon><FolderOpen size={34} /></EmptyIcon>No music sources configured.</EmptyState>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Issues to review</PanelTitle>
                <PanelDescription>Open a category to see the affected tracks and file paths.</PanelDescription>
              </div>
              <ButtonRow>
                <Button onClick={() => setActiveTab('library')}><ScanLine size={15} /> Library tools</Button>
                <Button onClick={() => setActiveTab('enrichment')}><Image size={15} /> Artwork tools</Button>
              </ButtonRow>
            </PanelHeader>
            <PanelBody>
              {diagnostics && (
                diagnostics.missing_files +
                diagnostics.duplicate_groups +
                diagnostics.invalid_metadata +
                diagnostics.unsupported_formats +
                diagnostics.missing_artwork
              ) > 0 ? (
                <IssueSections>
                  {[
                    { label: 'Missing audio files', count: diagnostics.missing_files, items: diagnostics.missing_file_details },
                    { label: 'Metadata issues', count: diagnostics.invalid_metadata, items: diagnostics.invalid_metadata_details },
                    { label: 'Unsupported formats', count: diagnostics.unsupported_formats, items: diagnostics.unsupported_format_details },
                    { label: 'Missing artwork', count: diagnostics.missing_artwork, items: diagnostics.missing_artwork_details },
                  ].filter(group => group.count > 0).map(group => (
                    <IssueGroup key={group.label}>
                      <summary>
                        <span>{group.label}</span>
                        <StatusBadge $status="running"><AlertTriangle size={14} /> {formatNumber(group.count)}</StatusBadge>
                      </summary>
                      <IssueList>
                        {group.items.map(item => (
                          <IssueRow key={`${group.label}-${item.id}`}>
                            <div>
                              <PrimaryText>{item.title || 'Untitled track'}</PrimaryText>
                              <SecondaryText>{item.artist || 'Unknown artist'}{item.album ? ` · ${item.album}` : ''}</SecondaryText>
                            </div>
                            <div>
                              <PrimaryText>{item.issue}</PrimaryText>
                              <IssuePath>{item.file_path || 'No file path stored'}</IssuePath>
                            </div>
                          </IssueRow>
                        ))}
                      </IssueList>
                    </IssueGroup>
                  ))}
                  {diagnostics.duplicate_groups > 0 && (
                    <IssueGroup>
                      <summary>
                        <span>Possible duplicate tracks</span>
                        <StatusBadge $status="running"><AlertTriangle size={14} /> {formatNumber(diagnostics.duplicate_groups)}</StatusBadge>
                      </summary>
                      <IssueList>
                        {diagnostics.duplicate_details.map(item => (
                          <IssueRow key={`${item.artist}-${item.title}`}>
                            <div>
                              <PrimaryText>{item.title || 'Untitled track'}</PrimaryText>
                              <SecondaryText>{item.artist || 'Unknown artist'} · {item.count} copies</SecondaryText>
                            </div>
                            <IssuePath>{item.paths.filter(Boolean).join('\n') || 'No file paths stored'}</IssuePath>
                          </IssueRow>
                        ))}
                      </IssueList>
                    </IssueGroup>
                  )}
                </IssueSections>
              ) : (
                <EmptyState>
                  <EmptyIcon><CheckCircle2 size={36} color={theme.colors.accent} /></EmptyIcon>
                  No library problems found.
                </EmptyState>
              )}
            </PanelBody>
          </Panel>

          <Panel>
            <PanelHeader>
              <div>
                <PanelTitle>Server health</PanelTitle>
                <PanelDescription>Live process and database information for troubleshooting.</PanelDescription>
              </div>
            </PanelHeader>
            <PanelBody>
              <SourceList>
                <SourceRow><span>Runtime</span><PrimaryText>{systemStatus?.go_version || '-'}</PrimaryText></SourceRow>
                <SourceRow><span>Background tasks</span><PrimaryText>{formatNumber(systemStatus?.goroutines)}</PrimaryText></SourceRow>
                <SourceRow><span>Database open / idle</span><PrimaryText>{systemStatus ? `${systemStatus.database_open} / ${systemStatus.database_idle}` : '-'}</PrimaryText></SourceRow>
                <SourceRow><span>Stored artwork files</span><PrimaryText>{formatNumber(systemStatus?.artwork_files)}</PrimaryText></SourceRow>
                <SourceRow>
                  <span>Automatic updates</span>
                  <StatusBadge $status={systemStatus?.automatic_updates ? 'completed' : 'failed'}>
                    {systemStatus?.automatic_updates ? <CheckCircle2 size={14} /> : <AlertTriangle size={14} />}
                    {systemStatus?.automatic_updates ? 'Available' : 'Unavailable'}
                  </StatusBadge>
                </SourceRow>
              </SourceList>
            </PanelBody>
          </Panel>
        </LibraryStack>
      )}

      {directoryPicker && (
        <PickerBackdrop role="presentation" onMouseDown={() => setDirectoryPicker(null)}>
          <PickerDialog
            role="dialog"
            aria-modal="true"
            aria-labelledby="folder-picker-title"
            onMouseDown={event => event.stopPropagation()}
          >
            <PickerHeader>
              <div>
                <PanelTitle id="folder-picker-title">Select music folder</PanelTitle>
                <PanelDescription>Browse folders available to the WaveNode server.</PanelDescription>
              </div>
              <DismissButton onClick={() => setDirectoryPicker(null)} aria-label="Close folder picker">
                <X size={19} />
              </DismissButton>
            </PickerHeader>

            <PickerPath title={directoryPicker.current_path}>{directoryPicker.current_path}</PickerPath>

            <PickerToolbar>
              {directoryPicker.parent_path && (
                <Button onClick={() => void browseDirectories(directoryPicker.parent_path)} disabled={directoryPickerLoading}>
                  <ArrowUp size={15} /> Up
                </Button>
              )}
              {directoryPicker.roots.map(root => (
                <Button
                  key={root}
                  onClick={() => void browseDirectories(root)}
                  disabled={directoryPickerLoading || root === directoryPicker.current_path}
                >
                  <HardDrive size={15} /> {root}
                </Button>
              ))}
            </PickerToolbar>

            <DirectoryList>
              {directoryPickerLoading ? (
                <EmptyState><Spin size={30} $active /> Opening folder...</EmptyState>
              ) : directoryPicker.directories.length ? (
                directoryPicker.directories.map(directory => (
                  <DirectoryButton key={directory.path} onClick={() => void browseDirectories(directory.path)}>
                    <FolderOpen size={19} color={theme.colors.accent} />
                    <span>{directory.name}</span>
                    <ChevronRight size={18} color="#777" />
                  </DirectoryButton>
                ))
              ) : (
                <EmptyState>This folder has no subfolders.</EmptyState>
              )}
            </DirectoryList>

            <PickerFooter>
              <Button onClick={() => setDirectoryPicker(null)}>Cancel</Button>
              <Button
                $variant="primary"
                onClick={() => void createMusicSource(directoryPicker.current_path)}
                disabled={sourceAction !== null || directoryPickerLoading}
              >
                <FolderOpen size={15} /> {sourceAction === 'add' ? 'Adding...' : 'Select this folder'}
              </Button>
            </PickerFooter>
          </PickerDialog>
        </PickerBackdrop>
      )}
    </Container>
  )
}

export default AdminDashboard
