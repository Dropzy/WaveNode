import { useEffect, useMemo, useState } from 'react'
import {
  ArrowLeft,
  ArrowRight,
  Check,
  ChevronRight,
  Folder,
  FolderOpen,
  HardDrive,
  Loader2,
  Music2,
  Plus,
  Server,
  ShieldCheck,
  Trash2,
  X,
} from 'lucide-react'
import styled from 'styled-components'
import {
  api,
  DirectoryBrowserData,
  ScanStatus,
  setupAPI,
  SetupStatus,
  tokenUtils,
} from '../services/api'

interface SetupProps {
  status: SetupStatus
}

type PickerTarget = 'music' | 'artwork'

const steps = ['Administrator', 'Music folders', 'Artwork', 'First scan']

const Setup = ({ status }: SetupProps) => {
  const [step, setStep] = useState(0)
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [musicPaths, setMusicPaths] = useState<string[]>([])
  const [artworkPath, setArtworkPath] = useState(status.default_artwork_path)
  const [pickerTarget, setPickerTarget] = useState<PickerTarget | null>(null)
  const [directoryData, setDirectoryData] = useState<DirectoryBrowserData | null>(null)
  const [pickerLoading, setPickerLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [scan, setScan] = useState<ScanStatus | null>(null)
  const [scanWarning, setScanWarning] = useState('')
  const [completed, setCompleted] = useState(false)

  const progress = useMemo(() => {
    if (!scan) return 0
    if (scan.total_files > 0) {
      return Math.min(100, Math.round((scan.processed / scan.total_files) * 100))
    }
    return Math.max(0, Math.min(100, scan.progress || 0))
  }, [scan])

  useEffect(() => {
    if (!scan?.id || completed || scanWarning) return

    const poll = window.setInterval(async () => {
      try {
        const response = await api.get<{ data?: ScanStatus | null }>('/scan/current')
        const currentScan = response.data.data
        if (currentScan) {
          setScan(currentScan)
          if (currentScan.status === 'completed' || currentScan.status === 'failed') {
            setCompleted(true)
          }
        } else {
          const historyResponse = await api.get<{ data?: ScanStatus[] }>('/admin/scans')
          const completedScan = historyResponse.data.data?.find(item => item.id === scan.id)
          if (completedScan) {
            setScan(completedScan)
          }
          setCompleted(true)
        }
      } catch {
        setScanWarning('The scan is still running, but live progress is temporarily unavailable.')
      }
    }, 1000)

    return () => window.clearInterval(poll)
  }, [scan?.id, completed, scanWarning])

  const browse = async (path?: string) => {
    setPickerLoading(true)
    setError('')
    try {
      setDirectoryData(await setupAPI.browseDirectories(path))
    } catch (err) {
      setError(readError(err, 'This folder cannot be opened.'))
    } finally {
      setPickerLoading(false)
    }
  }

  const openPicker = async (target: PickerTarget) => {
    setPickerTarget(target)
    setDirectoryData(null)
    await browse(target === 'artwork' && artworkPath ? artworkPath : undefined)
  }

  const chooseCurrentFolder = () => {
    if (!directoryData || !pickerTarget) return
    if (pickerTarget === 'music') {
      setMusicPaths(current =>
        current.includes(directoryData.current_path)
          ? current
          : [...current, directoryData.current_path],
      )
    } else {
      setArtworkPath(directoryData.current_path)
    }
    setPickerTarget(null)
  }

  const next = () => {
    setError('')
    if (step === 0) {
      if (username.trim().length < 3) {
        setError('Choose an administrator username with at least 3 characters.')
        return
      }
      if (!email.includes('@')) {
        setError('Enter a valid email address.')
        return
      }
      if (password.length < 8) {
        setError('Use at least 8 characters for the administrator password.')
        return
      }
      if (password !== confirmPassword) {
        setError('The passwords do not match.')
        return
      }
    }
    if (step === 1 && musicPaths.length === 0) {
      setError('Add at least one folder containing music.')
      return
    }
    if (step === 2 && !artworkPath.trim()) {
      setError('Choose a folder for artwork storage.')
      return
    }
    setStep(current => Math.min(steps.length - 1, current + 1))
  }

  const finishSetup = async () => {
    setSubmitting(true)
    setError('')
    try {
      const result = await setupAPI.complete({
        username: username.trim(),
        email: email.trim(),
        password,
        music_paths: musicPaths,
        artwork_path: artworkPath,
      })
      tokenUtils.setToken(result.token)
      setScan(result.scan)
      setScanWarning(result.scan_warning)
      if (!result.scan || result.scan.status !== 'running') {
        setCompleted(true)
      }
    } catch (err) {
      setError(readError(err, 'Setup could not be completed.'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Page>
      <Backdrop />
      <Shell>
        <Brand>
          <BrandMark><Music2 size={25} /></BrandMark>
          <div>
            <BrandName>WaveNode</BrandName>
            <BrandCaption>Private music, on your server</BrandCaption>
          </div>
        </Brand>

        <StepRail>
          {steps.map((label, index) => (
            <StepItem key={label} $active={index === step} $done={index < step}>
              <StepCircle>
                {index < step ? <Check size={17} /> : index + 1}
              </StepCircle>
              <span>{label}</span>
            </StepItem>
          ))}
        </StepRail>

        <Card>
          {step === 0 && (
            <StepContent>
              <IconPanel><ShieldCheck size={31} /></IconPanel>
              <Eyebrow>Welcome to WaveNode</Eyebrow>
              <Title>Create your administrator</Title>
              <Description>This account controls music folders, scans, users, and server settings.</Description>
              <FormGrid>
                <Field>
                  <label htmlFor="setup-username">Username</label>
                  <input id="setup-username" value={username} onChange={event => setUsername(event.target.value)} autoComplete="username" />
                </Field>
                <Field>
                  <label htmlFor="setup-email">Email</label>
                  <input id="setup-email" type="email" value={email} onChange={event => setEmail(event.target.value)} autoComplete="email" />
                </Field>
                <Field>
                  <label htmlFor="setup-password">Password</label>
                  <input id="setup-password" type="password" value={password} onChange={event => setPassword(event.target.value)} autoComplete="new-password" />
                </Field>
                <Field>
                  <label htmlFor="setup-confirm">Confirm password</label>
                  <input id="setup-confirm" type="password" value={confirmPassword} onChange={event => setConfirmPassword(event.target.value)} autoComplete="new-password" />
                </Field>
              </FormGrid>
            </StepContent>
          )}

          {step === 1 && (
            <StepContent>
              <IconPanel><FolderOpen size={31} /></IconPanel>
              <Eyebrow>Your library</Eyebrow>
              <Title>Add your music folders</Title>
              <Description>WaveNode scans folders visible to this server. You can add more later.</Description>
              <FolderList>
                {musicPaths.map(path => (
                  <FolderRow key={path}>
                    <Folder size={21} />
                    <PathText>{path}</PathText>
                    <IconButton type="button" aria-label={`Remove ${path}`} onClick={() => setMusicPaths(current => current.filter(item => item !== path))}>
                      <Trash2 size={18} />
                    </IconButton>
                  </FolderRow>
                ))}
                {musicPaths.length === 0 && <EmptyFolders>No music folders selected yet.</EmptyFolders>}
              </FolderList>
              <SecondaryButton type="button" onClick={() => void openPicker('music')}>
                <Plus size={18} /> Add music folder
              </SecondaryButton>
            </StepContent>
          )}

          {step === 2 && (
            <StepContent>
              <IconPanel><HardDrive size={31} /></IconPanel>
              <Eyebrow>Persistent storage</Eyebrow>
              <Title>Choose artwork storage</Title>
              <Description>Album covers and artist images are saved here so they survive restarts and upgrades.</Description>
              <SelectedPath>
                <Folder size={22} />
                <PathText>{artworkPath || 'No folder selected'}</PathText>
              </SelectedPath>
              <SecondaryButton type="button" onClick={() => void openPicker('artwork')}>
                <FolderOpen size={18} /> Choose artwork folder
              </SecondaryButton>
            </StepContent>
          )}

          {step === 3 && (
            <StepContent>
              <IconPanel>{completed ? <Check size={31} /> : <Server size={31} />}</IconPanel>
              <Eyebrow>{completed ? 'Ready to listen' : 'Final step'}</Eyebrow>
              <Title>{completed ? 'Your server is ready' : scan ? 'Building your library' : 'Start your first scan'}</Title>
              {!scan && !completed && (
                <>
                  <Description>We will save these settings, sign you in, and scan your folders for music.</Description>
                  <Summary>
                    <SummaryRow><span>Administrator</span><strong>{username}</strong></SummaryRow>
                    <SummaryRow><span>Music folders</span><strong>{musicPaths.length}</strong></SummaryRow>
                    <SummaryRow><span>Artwork folder</span><strong>{artworkPath}</strong></SummaryRow>
                  </Summary>
                </>
              )}
              {scan && (
                <ScanPanel>
                  <ProgressHeader>
                    <span>{scan.status === 'running' ? 'Scanning music files' : 'Library scan finished'}</span>
                    <strong>{progress}%</strong>
                  </ProgressHeader>
                  <ProgressTrack><ProgressFill style={{ width: `${progress}%` }} /></ProgressTrack>
                  <ScanDetail>
                    {scan.total_files > 0 ? `${scan.processed} of ${scan.total_files} files` : 'Finding music files...'}
                  </ScanDetail>
                  {scan.current_file && <CurrentFile>{scan.current_file}</CurrentFile>}
                </ScanPanel>
              )}
              {scanWarning && <Notice>{scanWarning}</Notice>}
              {completed && scan && (
                <ResultGrid>
                  <Result><strong>{scan.songs_added || 0}</strong><span>Tracks added</span></Result>
                  <Result><strong>{scan.songs_updated || 0}</strong><span>Tracks updated</span></Result>
                  <Result><strong>{scan.errors?.length || 0}</strong><span>Issues</span></Result>
                </ResultGrid>
              )}
            </StepContent>
          )}

          {error && <ErrorMessage>{error}</ErrorMessage>}

          <Actions>
            {step > 0 && !scan && (
              <BackButton type="button" onClick={() => { setError(''); setStep(current => current - 1) }}>
                <ArrowLeft size={18} /> Back
              </BackButton>
            )}
            <ActionSpacer />
            {step < steps.length - 1 && (
              <PrimaryButton type="button" onClick={next}>
                Continue <ArrowRight size={18} />
              </PrimaryButton>
            )}
            {step === steps.length - 1 && !scan && !completed && (
              <PrimaryButton type="button" disabled={submitting} onClick={() => void finishSetup()}>
                {submitting ? <><Loader2 className="spin" size={18} /> Saving...</> : <>Complete setup <ArrowRight size={18} /></>}
              </PrimaryButton>
            )}
            {step === steps.length - 1 && completed && (
              <PrimaryButton type="button" onClick={() => window.location.assign('/')}>
                Open WaveNode <ArrowRight size={18} />
              </PrimaryButton>
            )}
          </Actions>
        </Card>
      </Shell>

      {pickerTarget && (
        <ModalBackdrop role="presentation">
          <Modal role="dialog" aria-modal="true" aria-labelledby="folder-picker-title">
            <ModalHeader>
              <div>
                <h2 id="folder-picker-title">{pickerTarget === 'music' ? 'Choose a music folder' : 'Choose artwork storage'}</h2>
                <p>Select a folder on the server.</p>
              </div>
              <IconButton type="button" aria-label="Close folder picker" onClick={() => setPickerTarget(null)}><X size={20} /></IconButton>
            </ModalHeader>
            <LocationBar>
              <HardDrive size={18} />
              <PathText>{directoryData?.current_path || 'Loading folders...'}</PathText>
            </LocationBar>
            <RootButtons>
              {directoryData?.roots.map(root => (
                <RootButton key={root} type="button" onClick={() => void browse(root)}>{root}</RootButton>
              ))}
            </RootButtons>
            <DirectoryList>
              {pickerLoading && <PickerState><Loader2 className="spin" size={24} /> Opening folder...</PickerState>}
              {!pickerLoading && directoryData?.parent_path && (
                <DirectoryButton type="button" onClick={() => void browse(directoryData.parent_path)}>
                  <ArrowLeft size={19} /><span>Parent folder</span>
                </DirectoryButton>
              )}
              {!pickerLoading && directoryData?.directories.map(directory => (
                <DirectoryButton key={directory.path} type="button" onClick={() => void browse(directory.path)}>
                  <Folder size={20} /><span>{directory.name}</span><ChevronRight size={18} />
                </DirectoryButton>
              ))}
              {!pickerLoading && directoryData?.directories.length === 0 && <PickerState>This folder has no subfolders.</PickerState>}
            </DirectoryList>
            <ModalActions>
              <BackButton type="button" onClick={() => setPickerTarget(null)}>Cancel</BackButton>
              <PrimaryButton type="button" disabled={!directoryData} onClick={chooseCurrentFolder}>
                Use this folder <Check size={18} />
              </PrimaryButton>
            </ModalActions>
          </Modal>
        </ModalBackdrop>
      )}
    </Page>
  )
}

const readError = (error: unknown, fallback: string) => {
  const apiError = error as { response?: { data?: { error?: string } }; message?: string }
  return apiError.response?.data?.error || apiError.message || fallback
}

const Page = styled.main`
  min-height: 100vh; overflow: auto; position: relative; display: grid; place-items: center;
  padding: 42px 24px; background: #080d0a; color: #fff;
`
const Backdrop = styled.div`
  position: fixed; inset: 0; pointer-events: none;
  background: radial-gradient(circle at 50% -10%, rgba(30,215,96,.27), transparent 42%),
    linear-gradient(145deg, #0b1a10 0%, #080b09 55%, #111 100%);
`
const Shell = styled.div`position: relative; width: min(880px, 100%);`
const Brand = styled.div`display: flex; align-items: center; gap: 13px; margin-bottom: 26px;`
const BrandMark = styled.div`
  width: 48px; height: 48px; border-radius: 14px; display: grid; place-items: center;
  background: #1ed760; color: #07150c; box-shadow: 0 14px 35px rgba(30,215,96,.2);
`
const BrandName = styled.div`font-size: 22px; font-weight: 800; letter-spacing: -.4px;`
const BrandCaption = styled.div`font-size: 13px; color: #9aa39d; margin-top: 2px;`
const StepRail = styled.div`
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin: 0 4px 20px;
  @media (max-width: 650px) { grid-template-columns: repeat(2, 1fr); }
`
const StepItem = styled.div<{ $active: boolean; $done: boolean }>`
  display: flex; align-items: center; gap: 9px; min-width: 0; color: ${p => p.$active || p.$done ? '#fff' : '#727a75'};
  font-size: 13px; font-weight: 700;
`
const StepCircle = styled.div`
  width: 30px; height: 30px; flex: 0 0 auto; border-radius: 50%; display: grid; place-items: center;
  border: 1px solid currentColor; font-size: 12px;
  ${StepItem}[data-unused] & { background: transparent; }
`
const Card = styled.section`
  min-height: 510px; padding: 44px; border: 1px solid #29302b; border-radius: 22px;
  background: rgba(20,23,21,.94); box-shadow: 0 30px 80px rgba(0,0,0,.45);
  display: flex; flex-direction: column;
  @media (max-width: 650px) { padding: 28px 22px; min-height: 540px; }
`
const StepContent = styled.div`max-width: 690px; width: 100%; margin: 0 auto;`
const IconPanel = styled.div`
  width: 58px; height: 58px; display: grid; place-items: center; border-radius: 17px;
  color: #1ed760; background: rgba(30,215,96,.1); border: 1px solid rgba(30,215,96,.25); margin-bottom: 22px;
`
const Eyebrow = styled.div`color: #1ed760; text-transform: uppercase; letter-spacing: 1.5px; font-size: 12px; font-weight: 800;`
const Title = styled.h1`font-size: clamp(30px, 5vw, 42px); line-height: 1.08; letter-spacing: -1.2px; margin: 8px 0 12px;`
const Description = styled.p`color: #afb7b1; font-size: 16px; line-height: 1.55; max-width: 620px; margin-bottom: 28px;`
const FormGrid = styled.div`display: grid; grid-template-columns: 1fr 1fr; gap: 18px; @media (max-width: 650px) { grid-template-columns: 1fr; }`
const Field = styled.div`
  display: flex; flex-direction: column; gap: 8px;
  label { color: #dce1dd; font-size: 13px; font-weight: 700; }
  input { width: 100%; padding: 13px 14px; border-radius: 9px; border: 1px solid #3a403c; background: #0c0f0d; color: #fff; font-size: 15px; }
  input:focus { border-color: #1ed760; box-shadow: 0 0 0 3px rgba(30,215,96,.12); }
`
const FolderList = styled.div`display: grid; gap: 9px; margin-bottom: 18px; max-height: 220px; overflow: auto;`
const FolderRow = styled.div`display: flex; align-items: center; gap: 12px; padding: 13px 14px; border: 1px solid #303632; background: #0d100e; border-radius: 10px; color: #cfd5d1;`
const EmptyFolders = styled.div`padding: 28px; border: 1px dashed #3a403c; border-radius: 10px; color: #858d87; text-align: center;`
const PathText = styled.span`min-width: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;`
const IconButton = styled.button`color: #9da59f; padding: 7px; border-radius: 7px; &:hover { color: #fff; background: #292d2a; }`
const SelectedPath = styled.div`display: flex; gap: 12px; align-items: center; padding: 17px; margin-bottom: 18px; background: #0c0f0d; border: 1px solid #343a36; border-radius: 11px; color: #d9dedb;`
const Summary = styled.div`border: 1px solid #303632; border-radius: 12px; overflow: hidden;`
const SummaryRow = styled.div`
  display: grid; grid-template-columns: 150px 1fr; gap: 18px; padding: 14px 16px; border-bottom: 1px solid #292e2b;
  &:last-child { border-bottom: 0; } span { color: #909992; } strong { overflow-wrap: anywhere; }
`
const ScanPanel = styled.div`margin-top: 26px; padding: 20px; background: #0c100d; border: 1px solid #2e3731; border-radius: 13px;`
const ProgressHeader = styled.div`display: flex; justify-content: space-between; gap: 16px; font-weight: 700;`
const ProgressTrack = styled.div`height: 8px; background: #303532; border-radius: 999px; margin: 15px 0 10px; overflow: hidden;`
const ProgressFill = styled.div`height: 100%; background: #1ed760; border-radius: inherit; transition: width .3s ease;`
const ScanDetail = styled.div`font-size: 13px; color: #a8b0aa;`
const CurrentFile = styled.div`font-size: 12px; color: #747d76; margin-top: 8px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;`
const Notice = styled.div`margin-top: 18px; padding: 13px 15px; border-radius: 9px; color: #f2d77d; background: rgba(242,215,125,.08); border: 1px solid rgba(242,215,125,.2);`
const ResultGrid = styled.div`display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-top: 18px;`
const Result = styled.div`padding: 16px; border: 1px solid #303632; border-radius: 10px; strong { display: block; font-size: 25px; } span { color: #929b94; font-size: 12px; }`
const ErrorMessage = styled.div`max-width: 690px; width: 100%; margin: 20px auto 0; padding: 12px 14px; border-radius: 8px; color: #ffb4ab; background: rgba(244,67,54,.1); border: 1px solid rgba(244,67,54,.25);`
const Actions = styled.div`display: flex; align-items: center; gap: 12px; max-width: 690px; width: 100%; margin: auto auto 0; padding-top: 30px;`
const ActionSpacer = styled.div`flex: 1;`
const PrimaryButton = styled.button`
  min-height: 45px; display: inline-flex; align-items: center; justify-content: center; gap: 9px; padding: 0 19px;
  border-radius: 999px; background: #1ed760; color: #08140c; font-weight: 800;
  &:hover:not(:disabled) { background: #3be477; transform: translateY(-1px); }
  &:disabled { opacity: .55; cursor: wait; }
`
const SecondaryButton = styled(PrimaryButton)`background: #252a27; color: #fff; border: 1px solid #3b423e; &:hover:not(:disabled) { background: #303632; }`
const BackButton = styled.button`display: inline-flex; align-items: center; gap: 8px; color: #b5bdb7; font-weight: 700; padding: 10px; border-radius: 8px; &:hover { color: #fff; background: #242825; }`
const ModalBackdrop = styled.div`position: fixed; z-index: 100; inset: 0; display: grid; place-items: center; padding: 22px; background: rgba(0,0,0,.76); backdrop-filter: blur(7px);`
const Modal = styled.div`width: min(670px, 100%); max-height: min(720px, 90vh); display: flex; flex-direction: column; background: #151816; border: 1px solid #363d38; border-radius: 17px; box-shadow: 0 30px 100px #000;`
const ModalHeader = styled.div`display: flex; justify-content: space-between; gap: 20px; padding: 22px; border-bottom: 1px solid #2a302c; h2 { font-size: 20px; } p { color: #8f9891; font-size: 13px; margin-top: 5px; }`
const LocationBar = styled.div`display: flex; gap: 10px; align-items: center; margin: 16px 18px 9px; padding: 11px 13px; background: #0b0e0c; border: 1px solid #323833; border-radius: 8px;`
const RootButtons = styled.div`display: flex; flex-wrap: wrap; gap: 7px; padding: 0 18px 9px;`
const RootButton = styled.button`padding: 6px 10px; border: 1px solid #343a36; color: #cbd1cd; border-radius: 7px; &:hover { border-color: #1ed760; }`
const DirectoryList = styled.div`min-height: 240px; overflow: auto; padding: 8px 18px;`
const DirectoryButton = styled.button`width: 100%; display: flex; align-items: center; gap: 11px; padding: 12px; color: #e1e5e2; border-radius: 8px; text-align: left; span { flex: 1; } &:hover { background: #242925; }`
const PickerState = styled.div`min-height: 180px; display: flex; align-items: center; justify-content: center; gap: 9px; color: #8e9790;`
const ModalActions = styled.div`display: flex; justify-content: flex-end; gap: 12px; padding: 16px 18px; border-top: 1px solid #2a302c;`

export default Setup
