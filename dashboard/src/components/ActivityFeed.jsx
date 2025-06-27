import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Box,
  VStack,
  HStack,
  Text,
  Badge,
  Avatar,
  useColorModeValue,
  Circle,
  Spinner
} from '@chakra-ui/react'
import {
  FiDatabase,
  FiEdit,
  FiTrash2,
  FiPlus,
  FiUser,
  FiActivity,
  FiZap,
  FiAlertTriangle
} from 'react-icons/fi'

const MotionBox = motion(Box)

const getActivityIcon = (type) => {
  switch (type) {
    case 'create': return FiPlus
    case 'update': return FiEdit
    case 'delete': return FiTrash2
    case 'login': return FiUser
    case 'error': return FiAlertTriangle
    case 'request': return FiZap
    default: return FiActivity
  }
}

const getActivityColor = (type) => {
  switch (type) {
    case 'create': return 'green'
    case 'update': return 'blue'
    case 'delete': return 'red'
    case 'login': return 'purple'
    case 'error': return 'red'
    case 'request': return 'orange'
    default: return 'gray'
  }
}

// Mock activity data generator
const generateMockActivity = () => {
  const activities = [
    { type: 'create', action: 'Created new document', collection: 'users', user: 'admin', timestamp: new Date(Date.now() - Math.random() * 3600000) },
    { type: 'update', action: 'Updated document', collection: 'products', user: 'admin', timestamp: new Date(Date.now() - Math.random() * 3600000) },
    { type: 'delete', action: 'Deleted document', collection: 'todos', user: 'admin', timestamp: new Date(Date.now() - Math.random() * 3600000) },
    { type: 'login', action: 'Admin logged in', collection: null, user: 'admin', timestamp: new Date(Date.now() - Math.random() * 3600000) },
    { type: 'request', action: 'API request', collection: 'users', user: 'system', timestamp: new Date(Date.now() - Math.random() * 3600000) },
    { type: 'error', action: 'Failed to connect to database', collection: null, user: 'system', timestamp: new Date(Date.now() - Math.random() * 3600000) },
  ]
  
  return activities
    .sort((a, b) => b.timestamp - a.timestamp)
    .slice(0, 8)
    .map((activity, index) => ({ ...activity, id: index }))
}

export const ActivityFeed = ({ isLoading = false }) => {
  const [activities, setActivities] = useState([])
  const [isUpdating, setIsUpdating] = useState(false)
  
  const bgColor = useColorModeValue('white', 'gray.800')
  const borderColor = useColorModeValue('gray.200', 'gray.600')
  
  useEffect(() => {
    // Initial load
    setActivities(generateMockActivity())
    
    // Simulate real-time updates
    const interval = setInterval(() => {
      setIsUpdating(true)
      setTimeout(() => {
        setActivities(generateMockActivity())
        setIsUpdating(false)
      }, 500)
    }, 10000) // Update every 10 seconds
    
    return () => clearInterval(interval)
  }, [])
  
  const formatTime = (date) => {
    const now = new Date()
    const diff = now - date
    const minutes = Math.floor(diff / 60000)
    
    if (minutes < 1) return 'Just now'
    if (minutes < 60) return `${minutes}m ago`
    const hours = Math.floor(minutes / 60)
    if (hours < 24) return `${hours}h ago`
    return date.toLocaleDateString()
  }
  
  if (isLoading) {
    return (
      <VStack spacing={4} align="stretch">
        {[...Array(5)].map((_, i) => (
          <MotionBox
            key={i}
            p={4}
            bg={bgColor}
            borderRadius="lg"
            borderWidth="1px"
            borderColor={borderColor}
            animate={{ opacity: [1, 0.5, 1] }}
            transition={{ duration: 1.5, repeat: Infinity, delay: i * 0.2 }}
          >
            <HStack spacing={3}>
              <Circle size="40px" bg="gray.200" />
              <VStack align="start" spacing={1} flex={1}>
                <Box h="4" w="200px" bg="gray.200" borderRadius="md" />
                <Box h="3" w="120px" bg="gray.100" borderRadius="md" />
              </VStack>
            </HStack>
          </MotionBox>
        ))}
      </VStack>
    )
  }
  
  return (
    <VStack spacing={3} align="stretch" position="relative">
      {isUpdating && (
        <MotionBox
          position="absolute"
          top={-2}
          right={-2}
          initial={{ scale: 0 }}
          animate={{ scale: 1 }}
          exit={{ scale: 0 }}
        >
          <Circle size="20px" bg="blue.500">
            <Spinner size="xs" color="white" />
          </Circle>
        </MotionBox>
      )}
      
      <AnimatePresence>
        {activities.map((activity, index) => {
          const Icon = getActivityIcon(activity.type)
          const colorScheme = getActivityColor(activity.type)
          
          return (
            <MotionBox
              key={activity.id}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: 20 }}
              transition={{ duration: 0.3, delay: index * 0.1 }}
              whileHover={{ x: 4 }}
            >
              <HStack
                spacing={3}
                p={4}
                bg={bgColor}
                borderRadius="lg"
                borderWidth="1px"
                borderColor={borderColor}
                borderLeftWidth="4px"
                borderLeftColor={`${colorScheme}.400`}
                _hover={{
                  borderLeftColor: `${colorScheme}.500`,
                  boxShadow: 'md'
                }}
                transition="all 0.2s"
              >
                <Circle size="40px" bg={`${colorScheme}.100`} color={`${colorScheme}.600`}>
                  <Icon size={18} />
                </Circle>
                
                <VStack align="start" spacing={1} flex={1}>
                  <HStack spacing={2} wrap="wrap">
                    <Text fontWeight="medium" fontSize="sm">
                      {activity.action}
                    </Text>
                    {activity.collection && (
                      <Badge colorScheme={colorScheme} size="sm">
                        {activity.collection}
                      </Badge>
                    )}
                  </HStack>
                  
                  <HStack spacing={2} fontSize="xs" color="gray.500">
                    <Text>{activity.user}</Text>
                    <Text>•</Text>
                    <Text>{formatTime(activity.timestamp)}</Text>
                  </HStack>
                </VStack>
              </HStack>
            </MotionBox>
          )
        })}
      </AnimatePresence>
    </VStack>
  )
}

export default ActivityFeed