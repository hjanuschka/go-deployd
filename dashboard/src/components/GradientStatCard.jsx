import React from 'react'
import { motion } from 'framer-motion'
import {
  Box,
  Text,
  HStack,
  VStack,
  Icon,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  useColorModeValue
} from '@chakra-ui/react'
import { gradients } from '../theme/gradients'

const MotionBox = motion(Box)

export const GradientStatCard = ({ 
  title, 
  value, 
  helpText, 
  icon, 
  gradient = 'brand',
  delay = 0,
  trend = null,
  isLoading = false,
  onClick,
  subtitle
}) => {
  const textColor = useColorModeValue('white', 'white')
  const bg = useColorModeValue('white', 'gray.800')
  
  return (
    <MotionBox
      initial={{ opacity: 0, scale: 0.9 }}
      animate={{ opacity: 1, scale: 1 }}
      transition={{ duration: 0.6, delay }}
      whileHover={{ 
        scale: 1.02,
        boxShadow: '0 20px 40px -12px rgba(0, 0, 0, 0.25)'
      }}
      whileTap={{ scale: 0.98 }}
      onClick={onClick}
      cursor={onClick ? 'pointer' : 'default'}
    >
      <Box
        position="relative"
        borderRadius="xl"
        overflow="hidden"
        bg={bg}
        boxShadow="lg"
        _hover={{
          boxShadow: 'xl'
        }}
        transition="all 0.3s ease"
      >
        {/* Gradient Background */}
        <Box
          position="absolute"
          top={0}
          left={0}
          right={0}
          bottom={0}
          background={gradients[gradient]}
          opacity={0.9}
        />
        
        {/* Content */}
        <Box position="relative" p={6}>
          <HStack spacing={4} align="start">
            <MotionBox
              p={3}
              borderRadius="lg"
              bg="rgba(255,255,255,0.2)"
              backdropFilter="blur(10px)"
              initial={{ rotate: 0 }}
              whileHover={{ rotate: 360 }}
              transition={{ duration: 0.6 }}
            >
              <Icon as={icon} boxSize={8} color={textColor} />
            </MotionBox>
            
            <VStack align="start" spacing={1} flex={1}>
              <Text 
                fontSize="sm" 
                color="rgba(255,255,255,0.8)"
                fontWeight="medium"
                textTransform="uppercase"
                letterSpacing="wide"
              >
                {title}
              </Text>
              <Text 
                fontSize="3xl" 
                fontWeight="bold" 
                color={textColor}
                lineHeight="shorter"
              >
                {isLoading ? (
                  <MotionBox
                    width="60px"
                    height="32px"
                    bg="rgba(255,255,255,0.3)"
                    borderRadius="md"
                    animate={{ opacity: [1, 0.5, 1] }}
                    transition={{ duration: 1.5, repeat: Infinity }}
                  />
                ) : value}
              </Text>
              {(helpText || subtitle) && (
                <Text 
                  fontSize="xs" 
                  color="rgba(255,255,255,0.7)"
                  fontWeight="medium"
                >
                  {helpText || subtitle}
                </Text>
              )}
              {trend && (
                <Box>
                  <Text
                    fontSize="xs"
                    color={trend > 0 ? "green.300" : "red.300"}
                    fontWeight="semibold"
                  >
                    {trend > 0 ? '↗' : '↘'} {Math.abs(trend)}%
                  </Text>
                </Box>
              )}
            </VStack>
          </HStack>
        </Box>
        
        {/* Decorative Elements */}
        <Box
          position="absolute"
          top={-2}
          right={-2}
          width="60px"
          height="60px"
          borderRadius="full"
          bg="rgba(255,255,255,0.1)"
          transform="rotate(45deg)"
        />
        <Box
          position="absolute"
          bottom={-4}
          left={-4}
          width="40px"
          height="40px"
          borderRadius="full"
          bg="rgba(255,255,255,0.05)"
        />
      </Box>
    </MotionBox>
  )
}

export default GradientStatCard