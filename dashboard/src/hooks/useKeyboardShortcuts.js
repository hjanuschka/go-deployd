import { useEffect, useCallback } from 'react'
import { useToast } from '../components/ToastSystem'

export const useKeyboardShortcuts = () => {
  const { toast } = useToast()

  const shortcuts = {
    // Navigation shortcuts
    'cmd+k': {
      description: 'Open command palette',
      action: () => toast.info('Command palette coming soon!', { title: 'Feature Preview' })
    },
    'cmd+/': {
      description: 'Show keyboard shortcuts',
      action: () => showShortcutsHelp()
    },
    'cmd+r': {
      description: 'Refresh dashboard',
      action: () => window.location.reload()
    },
    // Dashboard shortcuts
    'g h': {
      description: 'Go to dashboard home',
      action: () => {
        if (window.location.pathname !== '/') {
          window.location.hash = '#/'
          toast.success('Navigated to dashboard home')
        }
      }
    },
    'g c': {
      description: 'Go to collections',
      action: () => toast.info('Collections page coming soon!', { title: 'Feature Preview' })
    },
    'g s': {
      description: 'Go to settings',
      action: () => toast.info('Settings page coming soon!', { title: 'Feature Preview' })
    },
    // Utility shortcuts
    'cmd+shift+d': {
      description: 'Toggle dark mode',
      action: () => {
        const event = new CustomEvent('toggleColorMode')
        window.dispatchEvent(event)
      }
    },
    'esc': {
      description: 'Close current modal/panel',
      action: () => {
        // Close any open modals or panels
        const modals = document.querySelectorAll('[role="dialog"]')
        if (modals.length > 0) {
          const closeButtons = document.querySelectorAll('[aria-label*="Close"]')
          if (closeButtons.length > 0) {
            closeButtons[closeButtons.length - 1].click()
          }
        }
      }
    }
  }

  const showShortcutsHelp = useCallback(() => {
    const shortcutsList = Object.entries(shortcuts)
      .map(([key, { description }]) => `${key}: ${description}`)
      .join('\n')
    
    toast.info(shortcutsList, {
      title: 'Keyboard Shortcuts',
      duration: 10000
    })
  }, [shortcuts])

  const handleKeyDown = useCallback((event) => {
    // Don't trigger shortcuts when typing in input fields
    if (event.target.tagName === 'INPUT' || event.target.tagName === 'TEXTAREA') {
      return
    }

    const key = []
    if (event.ctrlKey || event.metaKey) key.push('cmd')
    if (event.shiftKey) key.push('shift')
    if (event.altKey) key.push('alt')
    
    // Handle special keys
    if (event.key === 'Escape') {
      key.push('esc')
    } else if (event.key === '/') {
      key.push('/')
    } else if (event.key === 'r' && (event.ctrlKey || event.metaKey)) {
      key.push('r')
    } else if (event.key.length === 1) {
      key.push(event.key.toLowerCase())
    }
    
    const shortcutKey = key.join('+')
    
    // Handle special two-key sequences like "g h"
    if (window.lastKeyPressed === 'g' && event.key.length === 1) {
      const sequenceKey = `g ${event.key.toLowerCase()}`
      if (shortcuts[sequenceKey]) {
        event.preventDefault()
        shortcuts[sequenceKey].action()
        window.lastKeyPressed = null
        return
      }
    }
    
    // Store the last key for sequences
    if (event.key === 'g' && !event.ctrlKey && !event.metaKey) {
      window.lastKeyPressed = 'g'
      setTimeout(() => {
        window.lastKeyPressed = null
      }, 1000) // Reset after 1 second
      return
    }
    
    if (shortcuts[shortcutKey]) {
      event.preventDefault()
      shortcuts[shortcutKey].action()
    }
  }, [shortcuts])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    
    // Custom event listener for color mode toggle
    const handleColorModeToggle = () => {
      const colorModeButton = document.querySelector('[aria-label="Toggle color mode"]')
      if (colorModeButton) {
        colorModeButton.click()
        toast.success('Color mode toggled')
      }
    }
    
    window.addEventListener('toggleColorMode', handleColorModeToggle)
    
    return () => {
      window.removeEventListener('keydown', handleKeyDown)
      window.removeEventListener('toggleColorMode', handleColorModeToggle)
    }
  }, [handleKeyDown])

  return {
    shortcuts,
    showShortcutsHelp
  }
}

export default useKeyboardShortcuts