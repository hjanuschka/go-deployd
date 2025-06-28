import { useEffect, useRef, useState, useCallback } from 'react'

export const useWebSocket = (url, options = {}) => {
  const {
    onOpen,
    onClose,
    onMessage,
    onError,
    reconnect = true,
    reconnectInterval = 3000,
    reconnectAttempts = 5,
  } = options

  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState(null)
  const wsRef = useRef(null)
  const reconnectCountRef = useRef(0)
  const reconnectTimeoutRef = useRef(null)

  const connect = useCallback(() => {
    try {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        return
      }

      wsRef.current = new WebSocket(url)

      wsRef.current.onopen = (event) => {
        setIsConnected(true)
        reconnectCountRef.current = 0
        onOpen?.(event)
      }

      wsRef.current.onclose = (event) => {
        setIsConnected(false)
        onClose?.(event)

        // Attempt to reconnect if enabled
        if (
          reconnect &&
          reconnectCountRef.current < reconnectAttempts &&
          !event.wasClean
        ) {
          reconnectCountRef.current++
          reconnectTimeoutRef.current = setTimeout(() => {
            connect()
          }, reconnectInterval)
        }
      }

      wsRef.current.onmessage = (event) => {
        const data = JSON.parse(event.data)
        setLastMessage(data)
        onMessage?.(data, event)
      }

      wsRef.current.onerror = (event) => {
        onError?.(event)
      }
    } catch (error) {
      onError?.(error)
    }
  }, [url, onOpen, onClose, onMessage, onError, reconnect, reconnectInterval, reconnectAttempts])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    
    if (wsRef.current) {
      wsRef.current.close()
    }
  }, [])

  const sendMessage = useCallback((message) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    }
  }, [])

  useEffect(() => {
    connect()

    return () => {
      disconnect()
    }
  }, [connect, disconnect])

  return {
    isConnected,
    lastMessage,
    sendMessage,
    disconnect,
    reconnect: connect,
  }
}

export default useWebSocket