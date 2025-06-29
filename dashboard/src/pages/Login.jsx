import React, { useState } from 'react'
import { motion } from 'framer-motion'
import {
  Box,
  VStack,
  HStack,
  Heading,
  Text,
  Input,
  Button,
  Alert,
  AlertIcon,
  Container,
  Image,
  useColorModeValue,
  IconButton,
  useColorMode,
  FormControl,
  FormLabel,
  InputGroup,
  InputRightElement,
} from '@chakra-ui/react'
import { ViewIcon, ViewOffIcon, MoonIcon, SunIcon } from '@chakra-ui/icons'
import { AnimatedBackground } from '../components/AnimatedBackground'
import { gradients } from '../theme/gradients'

const MotionBox = motion(Box)
const MotionContainer = motion(Container)

function Login({ onLogin }) {
  const [masterKey, setMasterKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [showKey, setShowKey] = useState(false)
  const { colorMode, toggleColorMode } = useColorMode()

  const bg = useColorModeValue('white', 'gray.800')
  const borderColor = useColorModeValue('gray.200', 'gray.600')

  const handleLogin = async (e) => {
    e.preventDefault()
    setLoading(true)
    setError('')

    try {
      const result = await onLogin(masterKey)
      if (!result.success) {
        setError(result.message)
      }
    } catch (err) {
      setError('Failed to connect to server')
    } finally {
      setLoading(false)
    }
  }

  const containerVariants = {
    hidden: { opacity: 0, y: 50 },
    visible: {
      opacity: 1,
      y: 0,
      transition: {
        duration: 0.6,
        staggerChildren: 0.1
      }
    }
  }

  const itemVariants = {
    hidden: { opacity: 0, y: 20 },
    visible: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.4 }
    }
  }

  return (
    <Box position="relative" minH="100vh" overflow="hidden">
      <AnimatedBackground />
      
      {/* Color mode toggle */}
      <Box position="absolute" top={4} right={4} zIndex={3}>
        <IconButton
          onClick={toggleColorMode}
          variant="ghost"
          aria-label="Toggle color mode"
          icon={colorMode === 'light' ? <MoonIcon /> : <SunIcon />}
          bg="whiteAlpha.200"
          color="white"
          borderColor="whiteAlpha.300"
          _hover={{ bg: 'whiteAlpha.300' }}
        />
      </Box>

      <MotionContainer
        centerContent
        minH="100vh"
        py={8}
        position="relative"
        zIndex={2}
        variants={containerVariants}
        initial="hidden"
        animate="visible"
      >
        <VStack spacing={8} w="full" maxW="420px">
          {/* Logo and Title */}
          <MotionBox variants={itemVariants}>
            <VStack spacing={6}>
              <MotionBox
                boxSize="100px"
                whileHover={{ scale: 1.05, rotate: 5 }}
                whileTap={{ scale: 0.95 }}
                transition={{ type: "spring", stiffness: 300, damping: 20 }}
              >
                <img 
                  src="/_dashboard/deployd-logo.png" 
                  alt="Go-Deployd logo" 
                  style={{ 
                    width: '100%', 
                    height: '100%', 
                    objectFit: 'contain',
                    filter: 'drop-shadow(0 4px 8px rgba(0,0,0,0.3))'
                  }}
                />
              </MotionBox>
              <VStack spacing={3}>
                <Heading 
                  size="xl" 
                  textAlign="center" 
                  color={useColorModeValue('gray.800', 'white')}
                  bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
                  px={6}
                  py={3}
                  borderRadius="xl"
                  backdropFilter="blur(10px)"
                  textShadow={useColorModeValue('none', '2px 2px 4px rgba(0,0,0,0.5)')}
                  fontWeight="bold"
                >
                  Go-Deployd Dashboard
                </Heading>
                <Text 
                  color={useColorModeValue('gray.700', 'whiteAlpha.900')} 
                  textAlign="center"
                  fontSize="lg"
                  bg={useColorModeValue('whiteAlpha.800', 'blackAlpha.500')}
                  px={4}
                  py={2}
                  borderRadius="lg"
                  backdropFilter="blur(10px)"
                  textShadow={useColorModeValue('none', '1px 1px 2px rgba(0,0,0,0.3)')}
                >
                  Enter your master key to access the admin dashboard
                </Text>
              </VStack>
            </VStack>
          </MotionBox>

          {/* Enhanced Login Form */}
          <MotionBox
            w="full"
            variants={itemVariants}
            whileHover={{ y: -2 }}
            transition={{ type: "spring", stiffness: 300, damping: 20 }}
          >
            <Box
              w="full"
              p={8}
              bg={useColorModeValue('whiteAlpha.950', 'whiteAlpha.100')}
              backdropFilter="blur(20px)"
              borderWidth="1px"
              borderColor={useColorModeValue('gray.300', 'whiteAlpha.200')}
              borderRadius="2xl"
              shadow="2xl"
              position="relative"
              overflow="hidden"
              _before={{
                content: '""',
                position: 'absolute',
                top: 0,
                left: 0,
                right: 0,
                bottom: 0,
                background: gradients.brand,
                opacity: 0.1,
                zIndex: -1
              }}
            >
              <form onSubmit={handleLogin}>
                <VStack spacing={6}>
                  <FormControl isRequired>
                    <FormLabel color={useColorModeValue('gray.800', 'white')} fontWeight="semibold">
                      Master Key
                    </FormLabel>
                    <InputGroup>
                      <Input
                        type={showKey ? 'text' : 'password'}
                        value={masterKey}
                        onChange={(e) => setMasterKey(e.target.value)}
                        placeholder="mk_..."
                        fontFamily="mono"
                        fontSize="sm"
                        bg={useColorModeValue('white', 'whiteAlpha.100')}
                        borderColor={useColorModeValue('gray.300', 'whiteAlpha.300')}
                        color={useColorModeValue('gray.800', 'white')}
                        _placeholder={{ color: useColorModeValue('gray.500', 'whiteAlpha.600') }}
                        _hover={{ borderColor: useColorModeValue('gray.400', 'whiteAlpha.400') }}
                        _focus={{ 
                          borderColor: 'brand.400',
                          boxShadow: '0 0 0 1px var(--chakra-colors-brand-400)'
                        }}
                      />
                      <InputRightElement>
                        <IconButton
                          variant="ghost"
                          size="sm"
                          onClick={() => setShowKey(!showKey)}
                          icon={showKey ? <ViewOffIcon /> : <ViewIcon />}
                          aria-label={showKey ? 'Hide master key' : 'Show master key'}
                          color={useColorModeValue('gray.600', 'whiteAlpha.700')}
                          _hover={{ color: useColorModeValue('gray.800', 'white'), bg: useColorModeValue('gray.100', 'whiteAlpha.200') }}
                        />
                      </InputRightElement>
                    </InputGroup>
                  </FormControl>

                  {error && (
                    <MotionBox
                      initial={{ opacity: 0, scale: 0.9 }}
                      animate={{ opacity: 1, scale: 1 }}
                      w="full"
                    >
                      <Alert 
                        status="error" 
                        borderRadius="lg"
                        bg={useColorModeValue('red.50', 'red.500')}
                        color={useColorModeValue('red.800', 'white')}
                        border="1px solid"
                        borderColor={useColorModeValue('red.200', 'red.600')}
                      >
                        <AlertIcon color={useColorModeValue('red.800', 'white')} />
                        {error}
                      </Alert>
                    </MotionBox>
                  )}

                  <MotionBox
                    w="full"
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    <Button
                      type="submit"
                      size="lg"
                      w="full"
                      isLoading={loading}
                      loadingText="Authenticating..."
                      bg={gradients.brand}
                      color="white"
                      border="none"
                      _hover={{
                        bg: gradients.success,
                        transform: 'translateY(-1px)',
                        boxShadow: '0 8px 25px rgba(0,0,0,0.3)'
                      }}
                      _active={{
                        transform: 'translateY(0px)'
                      }}
                      boxShadow="0 4px 15px rgba(0,0,0,0.2)"
                      transition="all 0.2s"
                      fontWeight="bold"
                      fontSize="md"
                    >
                      Login to Dashboard
                    </Button>
                  </MotionBox>
                </VStack>
              </form>
            </Box>
          </MotionBox>

          {/* Enhanced Help Text */}
          <MotionBox variants={itemVariants}>
            <VStack 
              spacing={3} 
              textAlign="center"
              p={4}
              bg={useColorModeValue('whiteAlpha.900', 'whiteAlpha.100')}
              borderRadius="lg"
              backdropFilter="blur(10px)"
            >
              <Text fontSize="sm" color={useColorModeValue('gray.700', 'whiteAlpha.900')} fontWeight="medium">
                💡 The master key is displayed in the console when you first start the server.
              </Text>
              <Text fontSize="sm" color={useColorModeValue('gray.600', 'whiteAlpha.800')}>
                📁 It's also stored in <Text as="code" bg={useColorModeValue('gray.100', 'whiteAlpha.200')} px={2} py={1} borderRadius="md" color={useColorModeValue('gray.800', 'white')}>.deployd/security.json</Text>
              </Text>
            </VStack>
          </MotionBox>
        </VStack>
      </MotionContainer>
    </Box>
  )
}

export default Login