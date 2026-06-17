import { useEffect, useMemo, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import styled from 'styled-components'
import { ArrowLeft, Plus, Sparkles, Trash2 } from 'lucide-react'
import {
  playlistAPI,
  type Music,
  type SmartPlaylistCondition,
  type SmartPlaylistGroup,
  type SmartPlaylistRules,
} from '../services/api'
import { getTrackArtworkUrl } from '../utils/mediaUrl'

const fields = [
  ['title', 'Title'], ['artist', 'Artist'], ['album', 'Album'], ['genre', 'Genre'],
  ['year', 'Year'], ['duration', 'Duration (seconds)'], ['play_count', 'Play count'],
  ['rating', 'Rating'], ['date_added', 'Date added'], ['liked', 'Liked'],
  ['has_artwork', 'Has artwork'],
] as const

const textOperators = [['contains', 'contains'], ['not_contains', 'does not contain'], ['equals', 'is'], ['not_equals', 'is not']]
const numberOperators = [['equals', 'equals'], ['not_equals', 'does not equal'], ['greater_than', 'is greater than'], ['at_least', 'is at least'], ['less_than', 'is less than'], ['at_most', 'is at most']]
const dateOperators = [
  ['after', 'is after'], ['before', 'is before'],
  ['within_last_days', 'is within the last (days)'],
  ['not_within_last_days', 'is not within the last (days)'],
]
const booleanOperators = [['is_true', 'is true'], ['is_false', 'is false']]

const defaultRules: SmartPlaylistRules = {
  match: 'all',
  conditions: [{ field: 'genre', operator: 'contains', value: '' }],
  sort_by: 'date_added',
  sort_direction: 'desc',
  limit: 100,
}

function operatorsFor(field: SmartPlaylistCondition['field']) {
  if (['title', 'artist', 'album', 'genre'].includes(field)) return textOperators
  if (field === 'date_added') return dateOperators
  if (field === 'liked' || field === 'has_artwork') return booleanOperators
  return numberOperators
}

function defaultOperator(field: SmartPlaylistCondition['field']) {
  return operatorsFor(field)[0][0]
}

export default function SmartPlaylistEditor() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [rules, setRules] = useState<SmartPlaylistRules>(defaultRules)
  const [preview, setPreview] = useState<Music[]>([])
  const [previewLoaded, setPreviewLoaded] = useState(false)
  const [loading, setLoading] = useState(Boolean(id))
  const [previewing, setPreviewing] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!id) return
    void playlistAPI.getPlaylist(id)
      .then(playlist => {
        if (!playlist || playlist.type !== 'smart' || !playlist.smart_rules) {
          setError('Smart playlist not found.')
          return
        }
        setName(playlist.name)
        setDescription(playlist.description || '')
        setRules(playlist.smart_rules)
        setPreview([])
      })
      .catch(() => setError('Unable to load this smart playlist.'))
      .finally(() => setLoading(false))
  }, [id])

  useEffect(() => {
    if (id) return
    setName(searchParams.get('name') || '')
    setDescription(searchParams.get('description') || '')
  }, [id, searchParams])

  const hasIncompleteCondition = useMemo(
    () => {
      const incomplete = (conditions: SmartPlaylistCondition[], groups: SmartPlaylistGroup[] = []): boolean =>
        conditions.some(condition => !['liked', 'has_artwork'].includes(condition.field) && !condition.value.trim()) ||
        groups.some(group => incomplete(group.conditions, group.groups))
      return incomplete(rules.conditions, rules.groups)
    },
    [rules.conditions, rules.groups],
  )

  const updateCondition = (index: number, patch: Partial<SmartPlaylistCondition>) => {
    setRules(current => ({
      ...current,
      conditions: current.conditions.map((condition, conditionIndex) =>
        conditionIndex === index ? { ...condition, ...patch } : condition),
    }))
  }

  const addCondition = () => {
    setRules(current => ({
      ...current,
      conditions: [...current.conditions, { field: 'artist', operator: 'contains', value: '' }],
    }))
  }

  const updateGroupAtPath = (path: number[], updater: (group: SmartPlaylistGroup) => SmartPlaylistGroup) => {
    const updateGroups = (groups: SmartPlaylistGroup[], depth: number): SmartPlaylistGroup[] =>
      groups.map((group, index) => index === path[depth]
        ? depth === path.length - 1
          ? updater(group)
          : { ...group, groups: updateGroups(group.groups || [], depth + 1) }
        : group)
    setRules(current => ({ ...current, groups: updateGroups(current.groups || [], 0) }))
  }

  const addGroup = (parentPath?: number[]) => {
    const group: SmartPlaylistGroup = {
      match: 'all',
      conditions: [{ field: 'artist', operator: 'contains', value: '' }],
      groups: [],
    }
    if (!parentPath) {
      setRules(current => ({ ...current, groups: [...(current.groups || []), group] }))
      return
    }
    updateGroupAtPath(parentPath, current => ({ ...current, groups: [...(current.groups || []), group] }))
  }

  const removeGroup = (path: number[]) => {
    if (path.length === 1) {
      setRules(current => ({ ...current, groups: (current.groups || []).filter((_, index) => index !== path[0]) }))
      return
    }
    const parent = path.slice(0, -1)
    const index = path[path.length - 1]
    updateGroupAtPath(parent, group => ({ ...group, groups: (group.groups || []).filter((_, groupIndex) => groupIndex !== index) }))
  }

  const removeCondition = (index: number) => {
    setRules(current => ({
      ...current,
      conditions: current.conditions.filter((_, conditionIndex) => conditionIndex !== index),
    }))
  }

  const renderCondition = (
    condition: SmartPlaylistCondition,
    update: (patch: Partial<SmartPlaylistCondition>) => void,
    remove: () => void,
    key: string,
  ) => {
    const booleanField = condition.field === 'liked' || condition.field === 'has_artwork'
    const relativeDate = condition.field === 'date_added' && condition.operator.includes('within_last_days')
    return (
      <Rule key={key}>
        <Select value={condition.field} onChange={event => {
          const field = event.target.value as SmartPlaylistCondition['field']
          update({ field, operator: defaultOperator(field), value: field === 'date_added' ? new Date().toISOString().slice(0, 10) : '' })
        }}>
          {fields.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </Select>
        <Select value={condition.operator} onChange={event => update({
          operator: event.target.value,
          value: event.target.value.includes('within_last_days') ? '30' : condition.value,
        })}>
          {operatorsFor(condition.field).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </Select>
        {!booleanField && <Input
          type={relativeDate || ['year', 'duration', 'play_count', 'rating'].includes(condition.field) ? 'number' : condition.field === 'date_added' ? 'date' : 'text'}
          min={relativeDate ? 1 : undefined}
          value={condition.value}
          onChange={event => update({ value: event.target.value })}
          placeholder={relativeDate ? 'Days' : 'Value'}
        />}
        {booleanField && <span />}
        <IconButton onClick={remove} aria-label="Remove condition"><Trash2 size={17} /></IconButton>
      </Rule>
    )
  }

  const renderGroup = (group: SmartPlaylistGroup, path: number[], depth: number) => (
    <RuleGroup key={path.join('-')}>
      <GroupHeader>
        <div>Match <Select value={group.match} onChange={event => updateGroupAtPath(path, current => ({ ...current, match: event.target.value as 'all' | 'any' }))}><option value="all">all</option><option value="any">any</option></Select> in this group</div>
        <GroupActions>
          <SecondaryButton onClick={() => updateGroupAtPath(path, current => ({ ...current, conditions: [...current.conditions, { field: 'artist', operator: 'contains', value: '' }] }))}><Plus size={14} /> Condition</SecondaryButton>
          {depth < 3 && <SecondaryButton onClick={() => addGroup(path)}><Plus size={14} /> Nested group</SecondaryButton>}
          <IconButton onClick={() => removeGroup(path)} aria-label="Remove group"><Trash2 size={17} /></IconButton>
        </GroupActions>
      </GroupHeader>
      <Rules>
        {group.conditions.map((condition, index) => renderCondition(
          condition,
          patch => updateGroupAtPath(path, current => ({ ...current, conditions: current.conditions.map((item, itemIndex) => itemIndex === index ? { ...item, ...patch } : item) })),
          () => updateGroupAtPath(path, current => ({ ...current, conditions: current.conditions.filter((_, itemIndex) => itemIndex !== index) })),
          `${path.join('-')}-condition-${index}`,
        ))}
        {(group.groups || []).map((nested, index) => renderGroup(nested, [...path, index], depth + 1))}
      </Rules>
    </RuleGroup>
  )

  const loadPreview = async () => {
    setPreviewing(true)
    setPreviewLoaded(false)
    setError('')
    try {
      setPreview(await playlistAPI.previewSmartPlaylist(rules))
      setPreviewLoaded(true)
    } catch {
      setError('The preview could not be generated. Check the rule values.')
    } finally {
      setPreviewing(false)
    }
  }

  const save = async () => {
    if (!name.trim()) {
      setError('A playlist name is required.')
      return
    }
    setSaving(true)
    setError('')
    try {
      const payload = { name: name.trim(), description: description.trim(), smart_rules: rules }
      const playlist = id
        ? await playlistAPI.updateSmartPlaylist(id, payload)
        : await playlistAPI.createSmartPlaylist(payload)
      if (!playlist) throw new Error('No playlist returned')
      navigate(`/playlist/${playlist.id}`)
    } catch {
      setError('The smart playlist could not be saved. Check the rules and try again.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <Page><p>Loading smart playlist...</p></Page>

  return (
    <Page>
      <Back onClick={() => navigate(id ? `/playlist/${id}` : '/library')}>
        <ArrowLeft size={17} /> Back
      </Back>
      <Heading><Sparkles size={30} /> {id ? 'Edit smart playlist' : 'Create smart playlist'}</Heading>
      <Intro>WaveNode updates the tracks automatically. Subsonic players can browse and play the result as a read-only playlist.</Intro>

      <Panel>
        <FieldGrid>
          <Label>Playlist name<Input value={name} onChange={event => setName(event.target.value)} placeholder="Recently added favourites" /></Label>
          <Label>Description<Input value={description} onChange={event => setDescription(event.target.value)} placeholder="Optional" /></Label>
        </FieldGrid>

        <RuleHeading>
          <div>
            Match
            <Select value={rules.match} onChange={event => setRules(current => ({ ...current, match: event.target.value as 'all' | 'any' }))}>
              <option value="all">all conditions</option>
              <option value="any">any condition</option>
            </Select>
          </div>
          <GroupActions>
            <SecondaryButton onClick={addCondition}><Plus size={16} /> Add condition</SecondaryButton>
            <SecondaryButton onClick={() => addGroup()}><Plus size={16} /> Add group</SecondaryButton>
          </GroupActions>
        </RuleHeading>

        <Rules>
          {rules.conditions.length === 0 && <Muted>No conditions means every track in the library will match.</Muted>}
          {rules.conditions.map((condition, index) => renderCondition(
            condition,
            patch => updateCondition(index, patch),
            () => removeCondition(index),
            `root-${index}-${condition.field}`,
          ))}
          {(rules.groups || []).map((group, index) => renderGroup(group, [index], 1))}
        </Rules>

        <Options>
          <Label>Sort by<Select value={rules.sort_by} onChange={event => setRules(current => ({ ...current, sort_by: event.target.value as SmartPlaylistRules['sort_by'] }))}>
            <option value="date_added">Date added</option><option value="title">Title</option><option value="artist">Artist</option>
            <option value="album">Album</option><option value="genre">Genre</option><option value="year">Year</option>
            <option value="duration">Duration</option><option value="play_count">Play count</option><option value="rating">Rating</option>
            <option value="random">Random</option>
          </Select></Label>
          <Label>Direction<Select value={rules.sort_direction} disabled={rules.sort_by === 'random'} onChange={event => setRules(current => ({ ...current, sort_direction: event.target.value as 'asc' | 'desc' }))}>
            <option value="desc">Descending</option><option value="asc">Ascending</option>
          </Select></Label>
          <Label>Maximum tracks<Input type="number" min={1} max={500} value={rules.limit} onChange={event => setRules(current => ({ ...current, limit: Number(event.target.value) }))} /></Label>
        </Options>

        {error && <ErrorText>{error}</ErrorText>}
        <Actions>
          <SecondaryButton disabled={previewing || hasIncompleteCondition} onClick={() => void loadPreview()}>
            {previewing ? 'Building preview...' : 'Preview matches'}
          </SecondaryButton>
          <PrimaryButton disabled={saving || hasIncompleteCondition} onClick={() => void save()}>
            {saving ? 'Saving...' : id ? 'Save changes' : 'Create smart playlist'}
          </PrimaryButton>
        </Actions>
      </Panel>

      {previewLoaded && (
        <Panel>
          <PreviewTitle>{preview.length} matching tracks</PreviewTitle>
          {preview.length === 0 && <Muted>No tracks currently match these rules.</Muted>}
          {preview.slice(0, 20).map(track => (
            <PreviewTrack key={track.id}>
              <Artwork $url={getTrackArtworkUrl(track)} />
              <div><strong>{track.title}</strong><span>{track.artist} · {track.album}</span></div>
              <span>{Math.floor(track.duration / 60)}:{String(track.duration % 60).padStart(2, '0')}</span>
            </PreviewTrack>
          ))}
          {preview.length > 20 && <Muted>Showing the first 20 matches.</Muted>}
        </Panel>
      )}
    </Page>
  )
}

const Page = styled.main`padding: 28px clamp(18px, 3vw, 46px) 60px; overflow-y: auto; color: #fff;`
const Back = styled.button`display:flex;align-items:center;gap:7px;background:none;color:#b6bdb8;margin-bottom:20px;&:hover{color:#fff;}`
const Heading = styled.h1`display:flex;align-items:center;gap:12px;font-size:clamp(30px,4vw,48px);`
const Intro = styled.p`color:#b6bdb8;max-width:780px;margin:8px 0 24px;line-height:1.5;`
const Panel = styled.section`background:#151816;border:1px solid #2c322e;border-radius:14px;padding:22px;margin-bottom:20px;`
const FieldGrid = styled.div`display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px;@media(max-width:760px){grid-template-columns:1fr;}`
const Label = styled.label`display:grid;gap:7px;color:#cbd0cc;font-size:13px;font-weight:700;`
const Input = styled.input`width:100%;min-width:0;background:#242825;border:1px solid #414843;color:#fff;border-radius:8px;padding:11px 12px;&:focus{border-color:#1ed760;outline:none;}`
const Select = styled.select`background:#242825;border:1px solid #414843;color:#fff;border-radius:8px;padding:10px 12px;`
const RuleHeading = styled.div`display:flex;justify-content:space-between;align-items:center;gap:12px;margin:24px 0 12px;&>div{display:flex;align-items:center;gap:9px;font-weight:800;}`
const Rules = styled.div`display:grid;gap:9px;`
const Rule = styled.div`display:grid;grid-template-columns:1.1fr 1.2fr minmax(130px,2fr) 42px;gap:9px;@media(max-width:850px){grid-template-columns:1fr 1fr;}`
const RuleGroup = styled.div`display:grid;gap:10px;padding:14px;border:1px solid #39423c;border-left:3px solid #1ed760;border-radius:10px;background:#1b1f1c;`
const GroupHeader = styled.div`display:flex;justify-content:space-between;align-items:center;gap:12px;&>div{display:flex;align-items:center;gap:8px;font-weight:800;}@media(max-width:760px){align-items:flex-start;flex-direction:column;}`
const GroupActions = styled.div`display:flex;flex-wrap:wrap;gap:8px;`
const Options = styled.div`display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:14px;margin-top:22px;@media(max-width:700px){grid-template-columns:1fr;}`
const Actions = styled.div`display:flex;justify-content:flex-end;gap:10px;margin-top:22px;`
const SecondaryButton = styled.button`display:flex;align-items:center;justify-content:center;gap:7px;border:1px solid #59615b;color:#fff;border-radius:999px;padding:10px 16px;&:hover:not(:disabled){border-color:#fff;}&:disabled{opacity:.45;}`
const PrimaryButton = styled(SecondaryButton)`background:#1ed760;border-color:#1ed760;color:#07130b;font-weight:900;`
const IconButton = styled.button`display:grid;place-items:center;color:#d2d5d3;border-radius:8px;&:hover{background:#332224;color:#ff8b94;}`
const ErrorText = styled.p`color:#ff858d;margin-top:14px;`
const Muted = styled.p`color:#929a94;font-size:13px;`
const PreviewTitle = styled.h2`font-size:20px;margin-bottom:12px;`
const PreviewTrack = styled.div`display:grid;grid-template-columns:42px minmax(0,1fr) auto;align-items:center;gap:12px;padding:9px;border-bottom:1px solid #292e2a;& div{display:grid;gap:3px;min-width:0;}& strong,& div span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap;}& div span,&>span{color:#aeb5b0;font-size:12px;}`
const Artwork = styled.div<{ $url?: string }>`width:42px;height:42px;border-radius:5px;background:${({ $url }) => $url ? `url("${$url}") center/cover` : 'linear-gradient(135deg,#265f3b,#183020)'};`
