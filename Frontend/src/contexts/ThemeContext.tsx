import React, { createContext, useContext, useMemo, useState } from 'react'
import { ThemeProvider } from 'styled-components'
import {
  appThemes,
  defaultThemeName,
  isAppThemeName,
  themeStorageKey,
  type AppThemeName,
} from '../styles/themes'

interface ThemeContextValue {
  themeName: AppThemeName
  setThemeName: (themeName: AppThemeName) => void
}

const ThemeContext = createContext<ThemeContextValue | undefined>(undefined)

const getStoredTheme = (): AppThemeName => {
  if (typeof window === 'undefined') {
    return defaultThemeName
  }
  const stored = window.localStorage.getItem(themeStorageKey)
  return stored && isAppThemeName(stored) ? stored : defaultThemeName
}

export const AppThemeProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [themeName, setThemeNameState] = useState<AppThemeName>(getStoredTheme)

  const setThemeName = (nextThemeName: AppThemeName) => {
    setThemeNameState(nextThemeName)
    window.localStorage.setItem(themeStorageKey, nextThemeName)
  }

  const value = useMemo(() => ({ themeName, setThemeName }), [themeName])

  return (
    <ThemeContext.Provider value={value}>
      <ThemeProvider theme={appThemes[themeName]}>{children}</ThemeProvider>
    </ThemeContext.Provider>
  )
}

export const useAppTheme = () => {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useAppTheme must be used within AppThemeProvider')
  }
  return context
}
