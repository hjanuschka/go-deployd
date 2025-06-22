import React, { useState } from 'react'
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
      const response = await fetch('/_admin/auth/dashboard-login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ masterKey }),
      })

      const data = await response.json()

      if (data.success) {
        onLogin(masterKey)
      } else {
        setError(data.message || 'Invalid master key')
      }
    } catch (err) {
      setError('Failed to connect to server')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Container centerContent minH="100vh" py={8}>
      <Box position="absolute" top={4} right={4}>
        <IconButton
          onClick={toggleColorMode}
          variant="ghost"
          aria-label="Toggle color mode"
          icon={colorMode === 'light' ? <MoonIcon /> : <SunIcon />}
        />
      </Box>

      <VStack spacing={8} w="full" maxW="400px">
        {/* Logo and Title */}
        <VStack spacing={4}>
          <Image
            src="/deployd-icon.svg"
            alt="Go-Deployd"
            boxSize="80px"
            filter={colorMode === 'dark' ? 'invert(1)' : 'none'}
          />
          <VStack spacing={2}>
            <Heading size="xl" textAlign="center" color="brand.500">
              Go-Deployd Dashboard
            </Heading>
            <Text color="gray.500" textAlign="center">
              Enter your master key to access the admin dashboard
            </Text>
          </VStack>
        </VStack>

        {/* Login Form */}
        <Box
          w="full"
          p={8}
          bg={bg}
          borderWidth="1px"
          borderColor={borderColor}
          borderRadius="lg"
          shadow="lg"
        >
          <form onSubmit={handleLogin}>
            <VStack spacing={6}>
              <FormControl isRequired>
                <FormLabel>Master Key</FormLabel>
                <InputGroup>
                  <Input
                    type={showKey ? 'text' : 'password'}
                    value={masterKey}
                    onChange={(e) => setMasterKey(e.target.value)}
                    placeholder="mk_..."
                    fontFamily="mono"
                    fontSize="sm"
                  />
                  <InputRightElement>
                    <IconButton
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowKey(!showKey)}
                      icon={showKey ? <ViewOffIcon /> : <ViewIcon />}
                      aria-label={showKey ? 'Hide master key' : 'Show master key'}
                    />
                  </InputRightElement>
                </InputGroup>
              </FormControl>

              {error && (
                <Alert status="error" borderRadius="md">
                  <AlertIcon />
                  {error}
                </Alert>
              )}

              <Button
                type="submit"
                colorScheme="brand"
                size="lg"
                w="full"
                isLoading={loading}
                loadingText="Authenticating..."
              >
                Login to Dashboard
              </Button>
            </VStack>
          </form>
        </Box>

        {/* Help Text */}
        <VStack spacing={2} textAlign="center">
          <Text fontSize="sm" color="gray.500">
            The master key is displayed in the console when you first start the server.
          </Text>
          <Text fontSize="sm" color="gray.500">
            It's also stored in <code>.deployd/security.json</code>
          </Text>
        </VStack>
      </VStack>
    </Container>
  )
}

export default Login