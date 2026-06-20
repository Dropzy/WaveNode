export type AppThemeName = 'midnight' | 'dark' | 'ember' | 'daylight'

export interface AppTheme {
  name: AppThemeName
  label: string
  description: string
  colors: {
    background: string
    backgroundElevated: string
    surface: string
    surfaceSoft: string
    surfaceStrong: string
    text: string
    muted: string
    subtle: string
    border: string
    borderStrong: string
    accent: string
    accentHover: string
    accentSoft: string
    accentText: string
    accentGradient: string
    contentGradient: string
    playerBg: string
    playerGlow: string
    controlBg: string
    progressTrack: string
    danger: string
    success: string
    selection: string
    overlay: string
    shadow: string
  }
}

export const defaultThemeName: AppThemeName = 'midnight'
export const themeStorageKey = 'wavenode.theme'

export const appThemes: Record<AppThemeName, AppTheme> = {
  midnight: {
    name: 'midnight',
    label: 'Midnight Signal',
    description: 'Deep navy surfaces with clean cyan highlights.',
    colors: {
      background: '#070b16',
      backgroundElevated: '#0c1222',
      surface: '#111827',
      surfaceSoft: 'rgba(20, 30, 51, 0.72)',
      surfaceStrong: '#172033',
      text: '#f7fbff',
      muted: '#a9b7cc',
      subtle: '#718096',
      border: 'rgba(148, 163, 184, 0.18)',
      borderStrong: 'rgba(125, 211, 252, 0.34)',
      accent: '#38bdf8',
      accentHover: '#7dd3fc',
      accentSoft: 'rgba(56, 189, 248, 0.16)',
      accentText: '#031320',
      accentGradient: 'linear-gradient(135deg, #38bdf8 0%, #a78bfa 100%)',
      contentGradient: 'linear-gradient(to bottom, rgba(31, 74, 128, 0.72) 0, rgba(16, 27, 47, 0.52) 180px, #070b16 430px)',
      playerBg: 'linear-gradient(135deg, rgba(10, 18, 33, 0.96), rgba(15, 23, 42, 0.98))',
      playerGlow: 'rgba(56, 189, 248, 0.18)',
      controlBg: 'rgba(148, 163, 184, 0.10)',
      progressTrack: 'rgba(148, 163, 184, 0.24)',
      danger: '#fb7185',
      success: '#34d399',
      selection: 'rgba(56, 189, 248, 0.35)',
      overlay: 'rgba(3, 7, 18, 0.72)',
      shadow: 'rgba(0, 0, 0, 0.45)',
    },
  },
  dark: {
    name: 'dark',
    label: 'Dark Mode',
    description: 'Neutral charcoal surfaces with crisp blue-violet accents.',
    colors: {
      background: '#05070b',
      backgroundElevated: '#090d14',
      surface: '#111318',
      surfaceSoft: 'rgba(21, 24, 31, 0.78)',
      surfaceStrong: '#1a1d25',
      text: '#f5f7fb',
      muted: '#a8b0bd',
      subtle: '#697282',
      border: 'rgba(148, 163, 184, 0.14)',
      borderStrong: 'rgba(129, 140, 248, 0.34)',
      accent: '#60a5fa',
      accentHover: '#818cf8',
      accentSoft: 'rgba(96, 165, 250, 0.14)',
      accentText: '#05070b',
      accentGradient: 'linear-gradient(135deg, #60a5fa 0%, #8b5cf6 100%)',
      contentGradient: 'linear-gradient(to bottom, rgba(25, 35, 58, 0.76) 0, rgba(12, 18, 30, 0.58) 185px, #05070b 430px)',
      playerBg: 'linear-gradient(135deg, rgba(9, 13, 20, 0.98), rgba(15, 18, 27, 0.98))',
      playerGlow: 'rgba(96, 165, 250, 0.16)',
      controlBg: 'rgba(148, 163, 184, 0.08)',
      progressTrack: 'rgba(148, 163, 184, 0.20)',
      danger: '#fb7185',
      success: '#22c55e',
      selection: 'rgba(96, 165, 250, 0.30)',
      overlay: 'rgba(2, 6, 12, 0.76)',
      shadow: 'rgba(0, 0, 0, 0.58)',
    },
  },
  ember: {
    name: 'ember',
    label: 'Ember Studio',
    description: 'Warm graphite with amber and coral controls.',
    colors: {
      background: '#120d0a',
      backgroundElevated: '#1b130f',
      surface: '#211914',
      surfaceSoft: 'rgba(48, 32, 23, 0.74)',
      surfaceStrong: '#322117',
      text: '#fff8f1',
      muted: '#d4bca7',
      subtle: '#9f806a',
      border: 'rgba(251, 191, 36, 0.18)',
      borderStrong: 'rgba(251, 146, 60, 0.36)',
      accent: '#fb923c',
      accentHover: '#fbbf24',
      accentSoft: 'rgba(251, 146, 60, 0.16)',
      accentText: '#1f1006',
      accentGradient: 'linear-gradient(135deg, #f97316 0%, #facc15 100%)',
      contentGradient: 'linear-gradient(to bottom, rgba(103, 55, 25, 0.76) 0, rgba(52, 31, 20, 0.5) 180px, #120d0a 430px)',
      playerBg: 'linear-gradient(135deg, rgba(27, 19, 15, 0.96), rgba(38, 25, 18, 0.98))',
      playerGlow: 'rgba(251, 146, 60, 0.18)',
      controlBg: 'rgba(251, 191, 36, 0.10)',
      progressTrack: 'rgba(251, 191, 36, 0.22)',
      danger: '#f87171',
      success: '#86efac',
      selection: 'rgba(251, 146, 60, 0.35)',
      overlay: 'rgba(18, 13, 10, 0.74)',
      shadow: 'rgba(0, 0, 0, 0.48)',
    },
  },
  daylight: {
    name: 'daylight',
    label: 'Daylight',
    description: 'Bright, calm surfaces for daytime listening.',
    colors: {
      background: '#f3f7fb',
      backgroundElevated: '#ffffff',
      surface: '#ffffff',
      surfaceSoft: 'rgba(255, 255, 255, 0.78)',
      surfaceStrong: '#e9f0f7',
      text: '#111827',
      muted: '#475569',
      subtle: '#64748b',
      border: 'rgba(15, 23, 42, 0.12)',
      borderStrong: 'rgba(14, 116, 144, 0.28)',
      accent: '#0ea5e9',
      accentHover: '#0284c7',
      accentSoft: 'rgba(14, 165, 233, 0.14)',
      accentText: '#ffffff',
      accentGradient: 'linear-gradient(135deg, #0ea5e9 0%, #6366f1 100%)',
      contentGradient: 'linear-gradient(to bottom, rgba(186, 230, 253, 0.92) 0, rgba(224, 242, 254, 0.68) 190px, #f3f7fb 430px)',
      playerBg: 'linear-gradient(135deg, rgba(255, 255, 255, 0.97), rgba(232, 240, 249, 0.98))',
      playerGlow: 'rgba(14, 165, 233, 0.18)',
      controlBg: 'rgba(15, 23, 42, 0.06)',
      progressTrack: 'rgba(15, 23, 42, 0.16)',
      danger: '#dc2626',
      success: '#059669',
      selection: 'rgba(14, 165, 233, 0.26)',
      overlay: 'rgba(15, 23, 42, 0.42)',
      shadow: 'rgba(15, 23, 42, 0.18)',
    },
  },
}

export const isAppThemeName = (value: string): value is AppThemeName =>
  Object.prototype.hasOwnProperty.call(appThemes, value)
