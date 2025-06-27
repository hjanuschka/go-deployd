import React, { createContext, useContext, useState, useCallback } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Box,
  HStack,
  VStack,
  Text,
  IconButton,
  useColorModeValue,
  Portal,
  Icon,
} from '@chakra-ui/react'
import {
  FiX,
  FiCheck,
  FiInfo,
  FiAlertTriangle,
  FiAlertCircle,
} from 'react-icons/fi'
import { gradients } from '../theme/gradients'

const MotionBox = motion(Box)

const ToastContext = createContext()

export const useToast = () => {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useToast must be used within ToastProvider')
  }
  return context
}

export const ToastProvider = ({ children }) => {
  const [toasts, setToasts] = useState([])

  const addToast = useCallback((toast) => {
    const id = Date.now() + Math.random()
    const newToast = {
      id,
      type: 'info',
      duration: 5000,
      ...toast,
    }
    
    setToasts(prev => [...prev, newToast])
    
    if (newToast.duration > 0) {
      setTimeout(() => {
        removeToast(id)
      }, newToast.duration)
    }
    
    return id
  }, [])

  const removeToast = useCallback((id) => {
    setToasts(prev => prev.filter(toast => toast.id !== id))
  }, [])

  const toast = useCallback((message, options = {}) => {
    return addToast({ message, ...options })
  }, [addToast])

  toast.success = useCallback((message, options = {}) => {
    return addToast({ message, type: 'success', ...options })
  }, [addToast])

  toast.error = useCallback((message, options = {}) => {
    return addToast({ message, type: 'error', ...options })
  }, [addToast])

  toast.warning = useCallback((message, options = {}) => {
    return addToast({ message, type: 'warning', ...options })
  }, [addToast])

  toast.info = useCallback((message, options = {}) => {
    return addToast({ message, type: 'info', ...options })
  }, [addToast])

  const value = {
    toast,
    removeToast,
  }

  return (
    <ToastContext.Provider value={value}>
      {children}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  )
}

const ToastContainer = ({ toasts, onRemove }) => {
  return (
    <Portal>
      <Box
        position="fixed"
        top="20px"
        right="20px"
        zIndex="toast"
        pointerEvents="none"
      >
        <AnimatePresence>
          {toasts.map((toast) => (
            <ToastItem
              key={toast.id}
              toast={toast}
              onRemove={() => onRemove(toast.id)}
            />
          ))}
        </AnimatePresence>
      </Box>
    </Portal>
  )
}

const ToastItem = ({ toast, onRemove }) => {
  const getToastStyles = (type) => {
    switch (type) {
      case 'success':
        return {
          bg: gradients.success,
          icon: FiCheck,
          color: 'white'
        }
      case 'error':
        return {
          bg: gradients.danger,
          icon: FiAlertCircle,
          color: 'white'
        }
      case 'warning':
        return {
          bg: gradients.warning,
          icon: FiAlertTriangle,
          color: 'white'
        }
      default:
        return {
          bg: gradients.brand,
          icon: FiInfo,
          color: 'white'
        }
    }
  }

  const styles = getToastStyles(toast.type)

  return (
    <MotionBox
      initial={{ opacity: 0, x: 300, scale: 0.8 }}
      animate={{ opacity: 1, x: 0, scale: 1 }}
      exit={{ opacity: 0, x: 300, scale: 0.8 }}
      transition={{ duration: 0.3, type: "spring", stiffness: 300, damping: 30 }}
      mb={3}
      pointerEvents="auto"
    >
      <Box
        minW="320px"
        maxW="400px"
        p={4}
        bg={styles.bg}
        borderRadius="lg"
        shadow="2xl"
        borderWidth="1px"
        borderColor="whiteAlpha.200"
        backdropFilter="blur(10px)"
        position="relative"
        overflow="hidden"
        _before={{
          content: '""',
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          bg: 'whiteAlpha.100',
          zIndex: -1
        }}
      >
        <HStack spacing={3} align="start">
          <Box
            p={2}
            borderRadius="md"
            bg="whiteAlpha.200"
            color={styles.color}
            flexShrink={0}
          >
            <Icon as={styles.icon} boxSize={5} />
          </Box>
          
          <VStack align="start" spacing={1} flex={1} minW={0}>
            {toast.title && (
              <Text
                fontWeight="bold"
                fontSize="sm"
                color={styles.color}
                lineHeight="shorter"
              >
                {toast.title}
              </Text>
            )}
            <Text
              fontSize="sm"
              color={styles.color}
              lineHeight="base"
              wordBreak="break-word"
            >
              {toast.message}
            </Text>
          </VStack>

          <IconButton
            size="sm"
            variant="ghost"
            onClick={onRemove}
            icon={<FiX />}
            color={styles.color}
            _hover={{ bg: 'whiteAlpha.200' }}
            flexShrink={0}
            aria-label="Close toast"
          />
        </HStack>

        {/* Progress bar for timed toasts */}
        {toast.duration > 0 && (
          <MotionBox
            position="absolute"
            bottom={0}
            left={0}
            height="2px"
            bg="whiteAlpha.400"
            initial={{ width: '100%' }}
            animate={{ width: '0%' }}
            transition={{ duration: toast.duration / 1000, ease: 'linear' }}
          />
        )}
      </Box>
    </MotionBox>
  )
}

export default ToastProvider