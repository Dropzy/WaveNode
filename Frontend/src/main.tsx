import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App.tsx'
import { BrowserRouter, HashRouter } from 'react-router-dom'
import { initializeErrorHandling } from './utils/errorHandler'

// Initialize error handling to suppress autofill extension conflicts
initializeErrorHandling()

const isDesktop = Boolean((globalThis as { WAVENODE_DESKTOP?: boolean }).WAVENODE_DESKTOP)
const Router = isDesktop ? HashRouter : BrowserRouter

if (!isDesktop && 'serviceWorker' in navigator && import.meta.env.PROD) {
  window.addEventListener('load', () => {
    void navigator.serviceWorker.register('/sw.js')
  })
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <Router>
      <App />
    </Router>
  </React.StrictMode>,
)
