import { useCallback, useEffect, useMemo, useState } from 'react'
import { ArrowLeft, BookOpen, CheckCircle2, Play } from 'lucide-react'
import styled from 'styled-components'
import {
  audiobooksAPI,
  type AudiobookChapter,
  type AudiobookDetail,
  type AudiobookHome,
  type AudiobookProgress,
  type AudiobookSummary,
  type Music,
} from '../services/api'
import { useAudio } from '../contexts/AudioContext'
import { formatDuration } from '../utils/formatDuration'

const Section = styled.section`
  margin-bottom: 32px;
`

const SectionTitle = styled.h2`
  margin: 0 0 16px;
  color: ${({ theme }) => theme.colors.text};
  font-size: 22px;
`

const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 18px;

  @media (max-width: 520px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }
`

const BookCard = styled.button`
  min-width: 0;
  padding: 12px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: #181818;
  color: ${({ theme }) => theme.colors.text};
  text-align: left;
  cursor: pointer;

  &:hover, &:focus-visible {
    background: #242424;
    border-color: ${({ theme }) => theme.colors.borderStrong};
    outline: none;
  }
`

const Cover = styled.div<{ $image: string }>`
  width: 100%;
  aspect-ratio: 1;
  display: grid;
  place-items: center;
  margin-bottom: 10px;
  overflow: hidden;
  border-radius: 6px;
  background: ${({ $image }) => $image ? `url("${$image}") center / cover` : '#282828'};
  color: ${({ theme }) => theme.colors.muted};
`

const BookTitle = styled.div`
  overflow: hidden;
  color: ${({ theme }) => theme.colors.text};
  font-size: 14px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const Meta = styled.div`
  margin-top: 4px;
  overflow: hidden;
  color: ${({ theme }) => theme.colors.muted};
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
`

const ContinueRow = styled.div`
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(260px, 340px);
  gap: 14px;
  overflow-x: auto;
  padding-bottom: 8px;
`

const ContinueCard = styled.button`
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 12px;
  padding: 12px;
  border: 0;
  border-radius: 8px;
  background: #242424;
  color: ${({ theme }) => theme.colors.text};
  text-align: left;
  cursor: pointer;
`

const SmallCover = styled(Cover)`
  width: 72px;
  height: 72px;
  margin: 0;
`

const ProgressTrack = styled.div`
  height: 4px;
  margin-top: 9px;
  overflow: hidden;
  border-radius: 2px;
  background: #484848;
`

const ProgressFill = styled.div<{ $progress: number }>`
  width: ${({ $progress }) => Math.min(100, Math.max(0, $progress))}%;
  height: 100%;
  background: ${({ theme }) => theme.colors.accent};
`

const Message = styled.div`
  padding: 48px 12px;
  color: ${({ theme }) => theme.colors.muted};
  text-align: center;
`

const DetailHeader = styled.div`
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 24px;
  margin-bottom: 28px;

  @media (max-width: 640px) {
    grid-template-columns: 110px minmax(0, 1fr);
    gap: 16px;
  }
`

const DetailCover = styled(Cover)`
  margin: 0;
`

const BackButton = styled.button`
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
  padding: 8px 0;
  border: 0;
  background: transparent;
  color: ${({ theme }) => theme.colors.text};
  cursor: pointer;
`

const Description = styled.p`
  display: -webkit-box;
  margin: 12px 0 0;
  overflow: hidden;
  color: ${({ theme }) => theme.colors.muted};
  font-size: 13px;
  line-height: 1.5;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 5;
`

const ChapterList = styled.div`
  border-top: 1px solid ${({ theme }) => theme.colors.border};
`

const ChapterRow = styled.div`
  display: grid;
  grid-template-columns: 40px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  min-height: 68px;
  border-bottom: 1px solid ${({ theme }) => theme.colors.border};
`

const PlayButton = styled.button`
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 50%;
  background: #303030;
  color: ${({ theme }) => theme.colors.text};
  cursor: pointer;
`

const chapterTrack = (book: AudiobookSummary, chapter: AudiobookChapter): Music => ({
  id: `audiobook:${book.id}:${chapter.id}`,
  title: chapter.title || `Chapter ${chapter.number}`,
  artist: book.author,
  album: book.title,
  genre: 'Audiobook',
  duration: chapter.duration_seconds,
  release_date: '',
  file_path: '',
  image_url: book.image_url,
  created_at: '',
  updated_at: '',
  stream_url: chapter.audio_url,
  is_external: true,
  external_kind: 'audiobook',
  audiobook_id: book.id,
  audiobook_title: book.title,
  audiobook_author: book.author,
  audiobook_chapter_id: chapter.id,
  audiobook_chapter_number: chapter.number,
  audiobook_description: book.description,
  audiobook_website_url: book.website_url,
})

export function AudiobooksPanel({ query }: { query: string }) {
  const { playFromQueueAt } = useAudio()
  const [home, setHome] = useState<AudiobookHome>({ continue_listening: [], featured: [] })
  const [results, setResults] = useState<AudiobookSummary[]>([])
  const [detail, setDetail] = useState<AudiobookDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const loadHome = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const value = await audiobooksAPI.getHome()
      setHome(value)
      setResults(value.featured)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Audiobooks could not be loaded')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { void loadHome() }, [loadHome])

  useEffect(() => {
    const trimmed = query.trim()
    if (!trimmed) {
      setResults(home.featured)
      return
    }
    const timer = window.setTimeout(() => {
      setLoading(true)
      setError('')
      void audiobooksAPI.search(trimmed)
        .then(setResults)
        .catch(cause => setError(cause instanceof Error ? cause.message : 'Audiobook search failed'))
        .finally(() => setLoading(false))
    }, 350)
    return () => window.clearTimeout(timer)
  }, [home.featured, query])

  useEffect(() => {
    const update = (event: Event) => {
      const saved = (event as CustomEvent<AudiobookProgress>).detail
      setHome(previous => ({
        ...previous,
        continue_listening: saved.completed
          ? previous.continue_listening.filter(item => item.book_id !== saved.book_id)
          : [saved, ...previous.continue_listening.filter(item => item.book_id !== saved.book_id)].slice(0, 12),
      }))
      setDetail(previous => previous ? {
        ...previous,
        chapters: previous.chapters.map(chapter => chapter.id === saved.chapter_id
          ? { ...chapter, progress_seconds: saved.position_seconds, completed: saved.completed }
          : chapter),
      } : previous)
    }
    window.addEventListener('wavenode:audiobook-progress', update)
    return () => window.removeEventListener('wavenode:audiobook-progress', update)
  }, [])

  const openBook = useCallback(async (book: AudiobookSummary) => {
    setLoading(true)
    setError('')
    try {
      setDetail(await audiobooksAPI.get(book.id))
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Audiobook could not be loaded')
    } finally {
      setLoading(false)
    }
  }, [])

  const playChapter = useCallback((book: AudiobookSummary, chapters: AudiobookChapter[], index: number, position = 0) => {
    const queue = chapters.map(chapter => chapterTrack(book, chapter))
    playFromQueueAt(queue, index, position)
  }, [playFromQueueAt])

  const resume = useCallback(async (progress: AudiobookProgress) => {
    setLoading(true)
    try {
      const value = await audiobooksAPI.get(progress.book_id)
      setDetail(value)
      const index = Math.max(0, value.chapters.findIndex(chapter => chapter.id === progress.chapter_id))
      playChapter(value.book, value.chapters, index, progress.position_seconds)
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Audiobook could not be resumed')
    } finally {
      setLoading(false)
    }
  }, [playChapter])

  const visibleBooks = useMemo(() => results, [results])
  if (loading && !detail && visibleBooks.length === 0) return <Message>Loading audiobooks...</Message>
  if (error && !detail && visibleBooks.length === 0) return <Message>{error}</Message>

  if (detail) {
    return <div>
      <BackButton onClick={() => setDetail(null)}><ArrowLeft size={18} /> Audiobooks</BackButton>
      <DetailHeader>
        <DetailCover $image={detail.book.image_url}>{!detail.book.image_url && <BookOpen size={42} />}</DetailCover>
        <div>
          <h2>{detail.book.title}</h2>
          <Meta>{detail.book.author} · {detail.book.chapter_count} chapters · {formatDuration(detail.book.duration_seconds)}</Meta>
          <Description>{detail.book.description}</Description>
        </div>
      </DetailHeader>
      <ChapterList>
        {detail.chapters.map((chapter, index) => {
          const percent = chapter.duration_seconds > 0 ? chapter.progress_seconds / chapter.duration_seconds * 100 : 0
          return <ChapterRow key={chapter.id}>
            <PlayButton onClick={() => playChapter(detail.book, detail.chapters, index, chapter.completed ? 0 : chapter.progress_seconds)} aria-label={`Play ${chapter.title}`}><Play size={17} /></PlayButton>
            <div>
              <BookTitle>{chapter.number}. {chapter.title}</BookTitle>
              <Meta>{chapter.readers.join(', ') || detail.book.author}</Meta>
              {(chapter.progress_seconds > 0 || chapter.completed) && <ProgressTrack><ProgressFill $progress={chapter.completed ? 100 : percent} /></ProgressTrack>}
            </div>
            <Meta>{chapter.completed ? <CheckCircle2 size={18} /> : formatDuration(chapter.duration_seconds)}</Meta>
          </ChapterRow>
        })}
      </ChapterList>
    </div>
  }

  return <div>
    {error && <Message>{error}</Message>}
    {!query.trim() && home.continue_listening.length > 0 && <Section>
      <SectionTitle>Continue listening</SectionTitle>
      <ContinueRow>{home.continue_listening.map(item => {
        const percent = item.duration_seconds > 0 ? item.position_seconds / item.duration_seconds * 100 : 0
        return <ContinueCard key={`${item.book_id}:${item.chapter_id}`} onClick={() => void resume(item)}>
          <SmallCover $image={item.image_url}>{!item.image_url && <BookOpen />}</SmallCover>
          <div><BookTitle>{item.book_title}</BookTitle><Meta>{item.chapter_title}</Meta><ProgressTrack><ProgressFill $progress={percent} /></ProgressTrack></div>
        </ContinueCard>
      })}</ContinueRow>
    </Section>}
    <Section>
      <SectionTitle>{query.trim() ? 'Search results' : 'Featured public-domain audiobooks'}</SectionTitle>
      {loading ? <Message>Loading audiobooks...</Message> : visibleBooks.length === 0 ? <Message>No audiobooks found.</Message> : <Grid>
        {visibleBooks.map(book => <BookCard key={book.id} onClick={() => void openBook(book)}>
          <Cover $image={book.image_url}>{!book.image_url && <BookOpen size={42} />}</Cover>
          <BookTitle>{book.title}</BookTitle>
          <Meta>{book.author}</Meta>
          <Meta>{book.chapter_count} chapters · {formatDuration(book.duration_seconds)}</Meta>
        </BookCard>)}
      </Grid>}
    </Section>
  </div>
}
