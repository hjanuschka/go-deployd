import React, { createContext, useContext, useState, useEffect } from 'react'

const AuthContext = createContext({})

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}

export const AuthProvider = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false)
  const [masterKey, setMasterKey] = useState('')
  const [loading, setLoading] = useState(true)

  // Check for existing authentication on mount
  useEffect(() => {
    // Try to get master key from localStorage first
    const storedMasterKey = localStorage.getItem('masterKey')
    if (storedMasterKey) {
      setMasterKey(storedMasterKey)
      setIsAuthenticated(true)
      setLoading(false)
    } else {
      checkAuth()
    }
  }, [])

  const checkAuth = async () => {
    try {
      const response = await fetch('/_admin/auth/security-info', {
        credentials: 'include',
      })
      
      if (response.ok) {
        const data = await response.json()
        if (data.hasMasterKey) {
          setIsAuthenticated(true)
        }
      }
    } catch (error) {
      console.error('Auth check failed:', error)
    } finally {
      setLoading(false)
    }
  }

  const login = async (key) => {
    try {
      const response = await fetch('/_admin/auth/dashboard-login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        credentials: 'include',
        body: JSON.stringify({ masterKey: key }),
      })

      const data = await response.json()

      if (data.success) {
        setIsAuthenticated(true)
        setMasterKey(key)
        // Store master key in localStorage for persistence
        localStorage.setItem('masterKey', key)
        return { success: true }
      } else {
        return { success: false, message: data.message || 'Invalid master key' }
      }
    } catch (error) {
      return { success: false, message: 'Failed to connect to server' }
    }
  }

  const logout = () => {
    setIsAuthenticated(false)
    setMasterKey('')
    // Clear master key from localStorage
    localStorage.removeItem('masterKey')
    // Clear cookies by making a request to logout endpoint
    fetch('/_admin/auth/logout', {
      method: 'POST',
      credentials: 'include',
    }).catch(() => {
      // Ignore errors, just clear local state
    })
  }

  // Create authenticated fetch function that includes master key
  const authFetch = async (url, options = {}) => {
    const authOptions = {
      ...options,
      credentials: 'include',
      headers: {
        ...options.headers,
        ...(masterKey && { 'X-Master-Key': masterKey }),
      },
    }

    const response = await fetch(url, authOptions)
    
    // If we get 401, user needs to re-authenticate
    if (response.status === 401) {
      setIsAuthenticated(false)
      setMasterKey('')
    }
    
    return response
  }

  const value = {
    isAuthenticated,
    masterKey,
    loading,
    login,
    logout,
    authFetch,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export default AuthContext