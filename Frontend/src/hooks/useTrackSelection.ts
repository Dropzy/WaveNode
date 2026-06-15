import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type React from 'react'

type SelectableTrack = {
  id: string
}

type SelectionModifiers = Pick<React.MouseEvent, 'shiftKey' | 'ctrlKey' | 'metaKey'>

export const useTrackSelection = <T extends SelectableTrack>(tracks: T[]) => {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [anchorIndex, setAnchorIndex] = useState<number | null>(null)
  const rowRefs = useRef<Array<HTMLElement | null>>([])

  useEffect(() => {
    const availableIds = new Set(tracks.map(track => track.id))
    setSelectedIds(current => {
      const next = new Set([...current].filter(id => availableIds.has(id)))
      return next.size === current.size ? current : next
    })
    if (anchorIndex !== null && anchorIndex >= tracks.length) {
      setAnchorIndex(null)
    }
  }, [anchorIndex, tracks])

  useEffect(() => {
    const clearOnEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setSelectedIds(new Set())
        setAnchorIndex(null)
      }
    }
    document.addEventListener('keydown', clearOnEscape)
    return () => document.removeEventListener('keydown', clearOnEscape)
  }, [])

  const selectRange = useCallback((fromIndex: number, toIndex: number) => {
    const start = Math.min(fromIndex, toIndex)
    const end = Math.max(fromIndex, toIndex)
    setSelectedIds(new Set(tracks.slice(start, end + 1).map(track => track.id)))
  }, [tracks])

  const selectIndex = useCallback((index: number, event: SelectionModifiers) => {
    const track = tracks[index]
    if (!track) return

    if (event.shiftKey && anchorIndex !== null) {
      selectRange(anchorIndex, index)
      return
    }

    if (event.ctrlKey || event.metaKey) {
      setSelectedIds(current => {
        const next = new Set(current)
        if (next.has(track.id)) next.delete(track.id)
        else next.add(track.id)
        return next
      })
      setAnchorIndex(index)
      return
    }

    setSelectedIds(new Set([track.id]))
    setAnchorIndex(index)
  }, [anchorIndex, selectRange, tracks])

  const ensureSelected = useCallback((index: number) => {
    const track = tracks[index]
    if (!track || selectedIds.has(track.id)) return
    setSelectedIds(new Set([track.id]))
    setAnchorIndex(index)
  }, [selectedIds, tracks])

  const handleKeyDown = useCallback((
    index: number,
    event: React.KeyboardEvent<HTMLElement>,
    onActivate: () => void,
  ) => {
    if (event.key === 'Enter') {
      event.preventDefault()
      onActivate()
      return
    }
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return

    event.preventDefault()
    const nextIndex = Math.max(0, Math.min(tracks.length - 1, index + (event.key === 'ArrowDown' ? 1 : -1)))
    if (event.shiftKey) {
      selectRange(anchorIndex ?? index, nextIndex)
    } else {
      setSelectedIds(new Set([tracks[nextIndex].id]))
      setAnchorIndex(nextIndex)
    }
    rowRefs.current[nextIndex]?.focus()
  }, [anchorIndex, selectRange, tracks])

  const selectedTracks = useMemo(
    () => tracks.filter(track => selectedIds.has(track.id)),
    [selectedIds, tracks],
  )

  return {
    selectedIds,
    selectedTracks,
    rowRefs,
    selectIndex,
    ensureSelected,
    handleKeyDown,
  }
}
