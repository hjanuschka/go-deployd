import React from 'react'
import { motion } from 'framer-motion'
import {
  Grid,
  GridItem,
  Box,
  Text,
  VStack,
  Icon,
  useColorModeValue,
  HStack,
  Badge
} from '@chakra-ui/react'
import {
  FiPlus,
  FiDatabase,
  FiUsers,
  FiActivity,
  FiSettings,
  FiFileText,
  FiPlay,
  FiMonitor
} from 'react-icons/fi'
import { gradients } from '../theme/gradients'

const MotionBox = motion(Box)

const quickActions = [
  {
    title: 'Create Collection',
    description: 'Add a new data collection',
    icon: FiPlus,
    gradient: 'success',
    path: '/collections/new',
    badge: 'New'
  },
  {
    title: 'Manage Users',
    description: 'View and edit user accounts',
    icon: FiUsers,
    gradient: 'info',
    path: '/users'
  },
  {
    title: 'View Metrics',
    description: 'Analytics and performance',
    icon: FiActivity,
    gradient: 'warning',
    path: '/metrics'
  },
  {
    title: 'System Logs',
    description: 'Monitor system activity',
    icon: FiFileText,
    gradient: 'error',
    path: '/logs'
  },
  {
    title: 'API Testing',
    description: 'Test your API endpoints',
    icon: FiPlay,
    gradient: 'purple',
    path: '/self-test.html',
    external: true
  },
  {
    title: 'Settings',
    description: 'Configure your instance',
    icon: FiSettings,
    gradient: 'dark',
    path: '/settings'
  }
]

export const QuickActions = ({ onActionClick }) => {
  const bgColor = useColorModeValue('white', 'gray.800')
  const borderColor = useColorModeValue('gray.200', 'gray.600')
  
  const handleActionClick = (action) => {
    if (action.external) {
      window.open(action.path, '_blank')
    } else if (onActionClick) {
      onActionClick(action.path)
    }
  }
  
  return (
    <Grid 
      templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(3, 1fr)' }} 
      gap={4}
    >
      {quickActions.map((action, index) => (
        <GridItem key={action.title}>
          <MotionBox
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: index * 0.1 }}
            whileHover={{ 
              y: -8,
              transition: { duration: 0.2 }
            }}
            whileTap={{ scale: 0.95 }}
            onClick={() => handleActionClick(action)}
            cursor="pointer"
          >
            <Box
              p={6}
              bg={bgColor}
              borderRadius="xl"
              borderWidth="1px"
              borderColor={borderColor}
              position="relative"
              overflow="hidden"
              _hover={{
                boxShadow: 'xl',
                borderColor: 'transparent'
              }}
              transition="all 0.3s ease"
            >
              {/* Gradient Background Overlay */}
              <MotionBox
                position="absolute"
                top={0}
                left={0}
                right={0}
                bottom={0}
                background={gradients[action.gradient]}
                opacity={0}
                whileHover={{ opacity: 0.1 }}
                transition={{ duration: 0.3 }}
              />
              
              {/* Content */}
              <VStack spacing={4} align="start" position="relative">
                <HStack justify="space-between" w="full">
                  <MotionBox
                    p={3}
                    borderRadius="lg"
                    background={gradients[action.gradient]}
                    whileHover={{ scale: 1.1, rotate: 5 }}
                    transition={{ duration: 0.2 }}
                  >
                    <Icon as={action.icon} boxSize={6} color="white" />
                  </MotionBox>
                  
                  {action.badge && (
                    <Badge 
                      colorScheme="green" 
                      variant="subtle" 
                      fontSize="xs"
                      px={2}
                      py={1}
                      borderRadius="full"
                    >
                      {action.badge}
                    </Badge>
                  )}
                </HStack>
                
                <VStack align="start" spacing={2}>
                  <Text 
                    fontWeight="bold" 
                    fontSize="lg"
                    color={useColorModeValue('gray.800', 'white')}
                  >
                    {action.title}
                  </Text>
                  <Text 
                    fontSize="sm" 
                    color={useColorModeValue('gray.600', 'gray.300')}
                    lineHeight="tall"
                  >
                    {action.description}
                  </Text>
                </VStack>
              </VStack>
              
              {/* Decorative Elements */}
              <Box
                position="absolute"
                top={-10}
                right={-10}
                width="60px"
                height="60px"
                borderRadius="full"
                background={gradients[action.gradient]}
                opacity={0.1}
              />
              
              <Box
                position="absolute"
                bottom={-5}
                left={-5}
                width="30px"
                height="30px"
                borderRadius="full"
                background={gradients[action.gradient]}
                opacity={0.05}
              />
            </Box>
          </MotionBox>
        </GridItem>
      ))}
    </Grid>
  )
}

export default QuickActions