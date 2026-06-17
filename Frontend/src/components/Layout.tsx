import React, { useState } from 'react'
import styled from 'styled-components'
import { Sidebar } from './Sidebar'
import { Player } from './Player'
import { Menu, X } from 'lucide-react'

const LayoutContainer = styled.div`
  display: flex;
  height: 100vh;
  background-color: ${({ theme }) => theme.colors.background};
  position: relative;
`

const SkipLink = styled.a`
  position: fixed;
  top: 8px;
  left: 8px;
  z-index: 2000;
  padding: 10px 14px;
  border-radius: 6px;
  color: ${({ theme }) => theme.colors.accentText};
  background: ${({ theme }) => theme.colors.accentGradient};
  transform: translateY(-150%);

  &:focus {
    transform: translateY(0);
  }
`

const MainContent = styled.main`
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  
  @media (max-width: 768px) {
    margin-left: 0;
  }
`

const Content = styled.div`
  flex: 1;
  overflow-y: auto;
  background:
    ${({ theme }) => theme.colors.contentGradient},
    ${({ theme }) => theme.colors.background};
  
  @media (max-width: 768px) {
    padding-bottom: 90px; // Account for fixed player on mobile
  }
`

const MobileMenuButton = styled.button`
  display: none;
  position: fixed;
  top: 16px;
  left: 16px;
  z-index: 1001;
  background-color: ${({ theme }) => theme.colors.surface};
  border: none;
  border-radius: 50%;
  width: 48px;
  height: 48px;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s ease;
  
  &:hover {
    background-color: ${({ theme }) => theme.colors.surfaceStrong};
  }
  
  @media (max-width: 768px) {
    display: flex;
  }
  
  svg {
    width: 24px;
    height: 24px;
    color: ${({ theme }) => theme.colors.text};
  }
`

const Overlay = styled.div.withConfig({
  shouldForwardProp: (prop) => prop !== 'isOpen',
})<{ isOpen: boolean }>`
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: ${({ theme }) => theme.colors.overlay};
  z-index: 999;
  
  @media (max-width: 768px) {
    display: ${props => props.isOpen ? 'block' : 'none'};
  }
`

interface LayoutProps {
  children: React.ReactNode
}

export const Layout: React.FC<LayoutProps> = ({ children }) => {
  const [isSidebarOpen, setIsSidebarOpen] = useState(false)

  const toggleSidebar = () => {
    setIsSidebarOpen(!isSidebarOpen)
  }

  const closeSidebar = () => {
    setIsSidebarOpen(false)
  }

  return (
    <LayoutContainer>
      <SkipLink href="#main-content">Skip to main content</SkipLink>
      <MobileMenuButton onClick={toggleSidebar} aria-label={isSidebarOpen ? 'Close navigation' : 'Open navigation'}>
        {isSidebarOpen ? <X size={24} /> : <Menu size={24} />}
      </MobileMenuButton>
      
      <Sidebar isOpen={isSidebarOpen} onClose={closeSidebar} />
      
      <Overlay isOpen={isSidebarOpen} onClick={closeSidebar} />
      
      <MainContent id="main-content" tabIndex={-1}>
        <Content>{children}</Content>
        <Player />
      </MainContent>
    </LayoutContainer>
  )
}
