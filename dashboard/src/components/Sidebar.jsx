import React from 'react'
import {
  Box,
  VStack,
  HStack,
  Text,
  Icon,
  Button,
  Divider,
} from '@chakra-ui/react'
import { FiCode } from 'react-icons/fi'
import { useNavigate, useLocation } from 'react-router-dom'

function Sidebar({ menuItems, onClose }) {
  const navigate = useNavigate()
  const location = useLocation()

  const handleNavigation = (path) => {
    navigate(path)
    if (onClose) onClose()
  }

  return (
    <VStack h="full" spacing={0} align="stretch">
      {/* Logo */}
      <Box p={6}>
        <HStack spacing={3}>
          <Box boxSize={8}>
            <img 
              src="/deployd-logo.png" 
              alt="go-deployd logo" 
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          </Box>
          <Text fontSize="xl" fontWeight="bold" color="white">
            go-deployd
          </Text>
        </HStack>
      </Box>

      <Divider borderColor="gray.700" />

      {/* Navigation */}
      <VStack spacing={1} align="stretch" flex="1" p={4}>
        {menuItems.map((item) => {
          const isActive = location.pathname === item.path
          
          return (
            <Button
              key={item.path}
              variant="ghost"
              justifyContent="flex-start"
              leftIcon={<Icon as={item.icon} />}
              onClick={() => handleNavigation(item.path)}
              color={isActive ? 'white' : 'gray.300'}
              bg={isActive ? 'brand.500' : 'transparent'}
              _hover={{
                bg: isActive ? 'brand.600' : 'gray.700',
                color: 'white',
              }}
              _active={{
                bg: isActive ? 'brand.700' : 'gray.600',
              }}
              w="full"
              borderRadius="md"
            >
              {item.text}
            </Button>
          )
        })}
      </VStack>

      {/* Footer */}
      <Box p={4}>
        <Text fontSize="sm" color="gray.400" textAlign="center">
          v1.0.0
        </Text>
      </Box>
    </VStack>
  )
}

export default Sidebar