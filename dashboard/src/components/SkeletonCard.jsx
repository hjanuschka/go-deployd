import React from 'react'
import { motion } from 'framer-motion'
import {
  Box,
  VStack,
  HStack,
  Skeleton,
  SkeletonText,
  SkeletonCircle,
  useColorModeValue
} from '@chakra-ui/react'

const MotionBox = motion(Box)

export const SkeletonCard = ({ type = 'default', ...props }) => {
  const bgColor = useColorModeValue('white', 'gray.800')
  const borderColor = useColorModeValue('gray.200', 'gray.600')
  
  const pulseVariants = {
    initial: { opacity: 1 },
    animate: {
      opacity: [1, 0.5, 1],
      transition: {
        duration: 1.5,
        repeat: Infinity,
        ease: "easeInOut"
      }
    }
  }
  
  if (type === 'stat') {
    return (
      <MotionBox
        variants={pulseVariants}
        initial="initial"
        animate="animate"
        p={6}
        bg={bgColor}
        borderRadius="xl"
        borderWidth="1px"
        borderColor={borderColor}
        boxShadow="md"
        {...props}
      >
        <HStack spacing={4}>
          <SkeletonCircle size="16" />
          <VStack align="start" spacing={2} flex={1}>
            <Skeleton height="20px" width="100px" />
            <Skeleton height="32px" width="60px" />
            <Skeleton height="16px" width="80px" />
          </VStack>
        </HStack>
      </MotionBox>
    )
  }
  
  if (type === 'activity') {
    return (
      <MotionBox
        variants={pulseVariants}
        initial="initial"
        animate="animate"
        p={4}
        bg={bgColor}
        borderRadius="lg"
        borderWidth="1px"
        borderColor={borderColor}
        borderLeftWidth="4px"
        borderLeftColor="gray.300"
        {...props}
      >
        <HStack spacing={3}>
          <SkeletonCircle size="10" />
          <VStack align="start" spacing={1} flex={1}>
            <Skeleton height="16px" width="200px" />
            <Skeleton height="12px" width="120px" />
          </VStack>
        </HStack>
      </MotionBox>
    )
  }
  
  if (type === 'chart') {
    return (
      <MotionBox
        variants={pulseVariants}
        initial="initial"
        animate="animate"
        p={6}
        bg={bgColor}
        borderRadius="xl"
        borderWidth="1px"
        borderColor={borderColor}
        boxShadow="md"
        {...props}
      >
        <VStack spacing={4} align="stretch">
          <HStack justify="space-between">
            <VStack align="start" spacing={1}>
              <Skeleton height="24px" width="180px" />
              <Skeleton height="16px" width="250px" />
            </VStack>
            <Skeleton height="32px" width="100px" />
          </HStack>
          
          <Box h="300px" position="relative">
            <Skeleton height="100%" width="100%" borderRadius="md" />
            {/* Simulate chart bars */}
            <HStack 
              position="absolute" 
              bottom="20px" 
              left="20px" 
              right="20px" 
              justify="space-between"
              align="end"
            >
              {[...Array(7)].map((_, i) => (
                <Skeleton 
                  key={i}
                  height={`${Math.random() * 150 + 50}px`}
                  width="20px"
                  borderRadius="sm"
                />
              ))}
            </HStack>
          </Box>
        </VStack>
      </MotionBox>
    )
  }
  
  // Default card
  return (
    <MotionBox
      variants={pulseVariants}
      initial="initial"
      animate="animate"
      p={6}
      bg={bgColor}
      borderRadius="xl"
      borderWidth="1px"
      borderColor={borderColor}
      boxShadow="md"
      {...props}
    >
      <VStack spacing={4} align="stretch">
        <HStack spacing={3}>
          <SkeletonCircle size="12" />
          <VStack align="start" spacing={2} flex={1}>
            <Skeleton height="20px" width="150px" />
            <Skeleton height="16px" width="200px" />
          </VStack>
        </HStack>
        <SkeletonText mt="4" noOfLines={3} spacing="4" />
      </VStack>
    </MotionBox>
  )
}

export const SkeletonGrid = ({ type = 'default', count = 4, columns = 2 }) => {
  return (
    <Box 
      display="grid" 
      gridTemplateColumns={`repeat(${columns}, 1fr)`}
      gap={6}
    >
      {[...Array(count)].map((_, i) => (
        <SkeletonCard key={i} type={type} />
      ))}
    </Box>
  )
}

export default SkeletonCard