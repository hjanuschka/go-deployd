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
  const [token, setToken] = useState('')
  const [user, setUser] = useState(null)
  const [loading, setLoading] = useState(true)

  // Check for existing authentication on mount
  useEffect(() => {
    // Try to get JWT token from localStorage first
    const storedToken = localStorage.getItem('authToken')
    const storedUser = localStorage.getItem('authUser')
    if (storedToken && storedUser) {
      try {
        const userData = JSON.parse(storedUser)
        if (userData.isRoot) {
          setToken(storedToken)
          setUser(userData)
          setIsAuthenticated(true)
          setLoading(false)
          return
        }
      } catch (error) {
        // Clear invalid stored data
        localStorage.removeItem('authToken')
        localStorage.removeItem('authUser')
      }
    }
    checkAuth()
  }, [])

  const checkAuth = async () => {
    try {
      // Clear any invalid stored auth data
      localStorage.removeItem('authToken')
      localStorage.removeItem('authUser')
    } catch (error) {
      console.error('Auth check failed:', error)
    } finally {
      setLoading(false)
    }
  }

  const login = async (key) => {
    try {
      const response = await fetch('/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ masterKey: key }),
      })

      const data = await response.json()

      if (data.token && data.isRoot) {
        setIsAuthenticated(true)
        setToken(data.token)
        setUser({ isRoot: data.isRoot, ...data.user })
        // Store JWT token and user data in localStorage
        localStorage.setItem('authToken', data.token)
        localStorage.setItem('authUser', JSON.stringify({ isRoot: data.isRoot, ...data.user }))
        return { success: true }
      } else {
        return { success: false, message: data.error || 'Invalid master key or insufficient permissions' }
      }
    } catch (error) {
      return { success: false, message: 'Failed to connect to server' }
    }
  }

  const logout = () => {
    setIsAuthenticated(false)
    setToken('')
    setUser(null)
    // Clear JWT token and user data from localStorage
    localStorage.removeItem('authToken')
    localStorage.removeItem('authUser')
  }

  // Create authenticated fetch function that includes JWT token
  const authFetch = async (url, options = {}) => {
    const authOptions = {
      ...options,
      headers: {
        ...options.headers,
        ...(token && { 'Authorization': `Bearer ${token}` }),
      },
    }

    const response = await fetch(url, authOptions)
    
    // If we get 401, user needs to re-authenticate
    if (response.status === 401) {
      setIsAuthenticated(false)
      setToken('')
      setUser(null)
      localStorage.removeItem('authToken')
      localStorage.removeItem('authUser')
    }
    
    return response
  }

  const value = {
    isAuthenticated,
    token,
    user,
    loading,
    login,
    logout,
    authFetch,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export default AuthContext