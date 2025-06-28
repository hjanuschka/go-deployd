import React, { useState, useEffect } from 'react'
import {
  Box,
  VStack,
  HStack,
  Heading,
  Text,
  Card,
  CardBody,
  CardHeader,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Code,
  Badge,
  Button,
  Select,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Divider,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  useColorModeValue,
  useToast,
  IconButton,
  Tooltip,
  UnorderedList,
  ListItem,
  Link,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  useDisclosure,
} from '@chakra-ui/react'
import {
  FiCopy,
  FiPlay,
  FiBook,
  FiKey,
  FiUsers,
  FiDatabase,
  FiShield,
  FiCode,
  FiServer,
  FiEdit,
} from 'react-icons/fi'
import { useAuth } from '../contexts/AuthContext'
import { apiService } from '../services/api'
import { AnimatedBackground } from '../components/AnimatedBackground'

function Documentation() {
  const [collections, setCollections] = useState([])
  const [selectedCollection, setSelectedCollection] = useState('users')
  const [loading, setLoading] = useState(false)
  const [serverUrl, setServerUrl] = useState('')
  const [masterKey, setMasterKey] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [authMode, setAuthMode] = useState('masterkey') // 'masterkey' or 'userpass'
  const [lastResponse, setLastResponse] = useState(null)
  const [responseLoading, setResponseLoading] = useState(false)
  
  // Request editor state
  const { isOpen: isEditorOpen, onOpen: onEditorOpen, onClose: onEditorClose } = useDisclosure()
  const [editMethod, setEditMethod] = useState('GET')
  const [editUrl, setEditUrl] = useState('')
  const [editHeaders, setEditHeaders] = useState('{}')
  const [editBody, setEditBody] = useState('')
  
  const { authFetch } = useAuth()
  const toast = useToast()
  const cardBg = useColorModeValue('white', 'gray.700')
  const codeBg = useColorModeValue('gray.50', 'gray.800')

  useEffect(() => {
    loadCollections()
    setServerUrl(window.location.origin)
    loadMasterKey()
  }, [])

  const loadCollections = async () => {
    try {
      setLoading(true)
      const response = await authFetch('/_admin/collections')
      if (response.ok) {
        const data = await response.json()
        setCollections(data || [])
        if (data && data.length > 0) {
          setSelectedCollection(data[0].name)
        }
      }
    } catch (err) {
      console.error('Failed to load collections:', err)
    } finally {
      setLoading(false)
    }
  }

  const loadMasterKey = async () => {
    try {
      const response = await authFetch('/_admin/auth/security-info')
      if (response.ok) {
        const data = await response.json()
        // Don't expose the actual master key, just indicate it exists
        setMasterKey('mk_your_master_key_here')
      }
    } catch (err) {
      console.error('Failed to load security info:', err)
    }
  }

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text).then(() => {
      toast({
        title: 'Copied to clipboard',
        status: 'success',
        duration: 2000,
        isClosable: true,
      })
    })
  }

  const executeDocRequest = async (codeContent) => {
    try {
      setResponseLoading(true)
      setLastResponse(null)
      
      // Parse curl command to extract method, url, headers, and body
      const lines = codeContent.split('\n').map(line => line.trim()).filter(line => line)
      const curlLine = lines.find(line => line.startsWith('curl'))
      
      if (!curlLine) {
        toast({
          title: 'Invalid Request',
          description: 'Could not parse curl command from code block',
          status: 'error',
          duration: 3000,
          isClosable: true,
        })
        setResponseLoading(false)
        return
      }

      // Extract method
      const methodMatch = curlLine.match(/-X\s+(\w+)/)
      const method = methodMatch ? methodMatch[1] : 'GET'

      // Extract URL
      const urlMatch = curlLine.match(/"([^"]*)"(?:\s|$)/) || curlLine.match(/\s([^\s-][^\s]*?)(?:\s|$)/)
      let url = urlMatch ? urlMatch[1] : '/'
      
      // Replace variables in URL
      url = url.replace(/\$\{serverUrl\}/g, window.location.origin)
      url = url.replace(/\$\{masterKey\}/g, masterKey || 'your_master_key_here')

      // Extract headers
      const headers = {}
      const headerLines = lines.filter(line => line.includes('-H'))
      headerLines.forEach(line => {
        const headerMatch = line.match(/-H\s+"([^:]+):\s*([^"]+)"/)
        if (headerMatch) {
          headers[headerMatch[1]] = headerMatch[2].replace(/\$\{masterKey\}/g, masterKey || 'your_master_key_here')
        }
      })

      // Add authentication headers based on current auth mode
      if (authMode === 'masterkey' && masterKey) {
        headers['X-Master-Key'] = masterKey
      } else if (authMode === 'userpass' && username && password) {
        // For user/pass, we could add basic auth or custom headers
        headers['Authorization'] = `Basic ${btoa(`${username}:${password}`)}`
      }

      // Extract body
      let body = null
      const bodyLines = lines.filter(line => line.includes('-d'))
      if (bodyLines.length > 0) {
        const bodyContent = bodyLines.join(' ').match(/-d\s+'([^']+)'||-d\s+"([^"]+)"/)
        if (bodyContent) {
          body = bodyContent[1] || bodyContent[2]
          try {
            body = JSON.parse(body)
          } catch (e) {
            // Keep as string if not valid JSON
          }
        }
      }

      const startTime = Date.now()
      
      // Execute the request
      const response = await apiService.testEndpoint(method, url, body, headers)
      const endTime = Date.now()
      
      const responseData = {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers && typeof response.headers.entries === 'function' 
          ? Object.fromEntries(response.headers.entries()) 
          : response.headers || {},
        data: response.data,
        duration: endTime - startTime,
        timestamp: new Date().toISOString(),
        method,
        url
      }

      setLastResponse(responseData)
      console.log('Setting response data:', responseData)
      
      toast({
        title: 'Request Executed',
        description: `${method} ${url} - ${response.status} ${response.statusText}`,
        status: response.status >= 200 && response.status < 300 ? 'success' : 'error',
        duration: 3000,
        isClosable: true,
      })
      
    } catch (error) {
      const errorResponse = {
        error: true,
        message: error.message,
        duration: Date.now() - (typeof startTime !== 'undefined' ? startTime : Date.now()),
        timestamp: new Date().toISOString(),
        method: typeof method !== 'undefined' ? method : 'Unknown',
        url: typeof url !== 'undefined' ? url : 'Unknown'
      }
      
      if (error.response) {
        errorResponse.status = error.response.status
        errorResponse.statusText = error.response.statusText
        errorResponse.data = error.response.data
      }
      
      setLastResponse(errorResponse)
      
      toast({
        title: 'Request Failed',
        description: error.message,
        status: 'error',
        duration: 5000,
        isClosable: true,
      })
    } finally {
      setResponseLoading(false)
    }
  }

  const openRequestEditor = (codeContent) => {
    try {
      // Parse curl command to pre-populate the editor
      const lines = codeContent.split('\n').map(line => line.trim()).filter(line => line)
      const curlLine = lines.find(line => line.startsWith('curl'))
      
      if (!curlLine) return

      // Extract method
      const methodMatch = curlLine.match(/-X\s+(\w+)/)
      const method = methodMatch ? methodMatch[1] : 'GET'
      setEditMethod(method)

      // Extract URL
      const urlMatch = curlLine.match(/"([^"]*)"(?:\s|$)/) || curlLine.match(/\s([^\s-][^\s]*?)(?:\s|$)/)
      let url = urlMatch ? urlMatch[1] : '/'
      
      // Replace variables in URL
      url = url.replace(/\$\{serverUrl\}/g, window.location.origin)
      url = url.replace(/\$\{masterKey\}/g, masterKey || 'your_master_key_here')
      setEditUrl(url)

      // Extract headers
      const headers = {}
      const headerLines = lines.filter(line => line.includes('-H'))
      headerLines.forEach(line => {
        const headerMatch = line.match(/-H\s+"([^:]+):\s*([^"]+)"/)
        if (headerMatch) {
          headers[headerMatch[1]] = headerMatch[2].replace(/\$\{masterKey\}/g, masterKey || 'your_master_key_here')
        }
      })

      // Add authentication headers based on current auth mode
      if (authMode === 'masterkey' && masterKey) {
        headers['X-Master-Key'] = masterKey
      } else if (authMode === 'userpass' && username && password) {
        headers['Authorization'] = `Basic ${btoa(`${username}:${password}`)}`
      }

      setEditHeaders(JSON.stringify(headers, null, 2))

      // Extract body
      let body = ''
      const bodyLines = lines.filter(line => line.includes('-d'))
      if (bodyLines.length > 0) {
        const bodyContent = bodyLines.join(' ').match(/-d\s+'([^']+)'||-d\s+"([^"]+)"/)
        if (bodyContent) {
          body = bodyContent[1] || bodyContent[2]
          try {
            body = JSON.stringify(JSON.parse(body), null, 2)
          } catch (e) {
            // Keep as string if not valid JSON
          }
        }
      }
      setEditBody(body)

      onEditorOpen()
    } catch (error) {
      console.error('Failed to parse curl command:', error)
      toast({
        title: 'Parse Error',
        description: 'Could not parse the curl command for editing',
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const executeEditedRequest = async () => {
    try {
      setResponseLoading(true)
      setLastResponse(null)

      // Parse headers
      let parsedHeaders = {}
      if (editHeaders.trim()) {
        try {
          parsedHeaders = JSON.parse(editHeaders)
        } catch (err) {
          throw new Error('Invalid JSON in headers')
        }
      }

      // Parse body for non-GET requests
      let parsedBody = null
      if (editMethod !== 'GET' && editBody.trim()) {
        if (parsedHeaders['Content-Type']?.includes('application/json')) {
          try {
            parsedBody = JSON.parse(editBody)
          } catch (err) {
            throw new Error('Invalid JSON in request body')
          }
        } else {
          parsedBody = editBody
        }
      }

      const startTime = Date.now()
      
      // Execute the request
      const response = await apiService.testEndpoint(editMethod, editUrl, parsedBody, parsedHeaders)
      const endTime = Date.now()
      
      const responseData = {
        status: response.status,
        statusText: response.statusText,
        headers: response.headers && typeof response.headers.entries === 'function' 
          ? Object.fromEntries(response.headers.entries()) 
          : response.headers || {},
        data: response.data,
        duration: endTime - startTime,
        timestamp: new Date().toISOString(),
        method: editMethod,
        url: editUrl
      }

      setLastResponse(responseData)
      onEditorClose()
      
      toast({
        title: 'Request Executed',
        description: `${editMethod} ${editUrl} - ${response.status} ${response.statusText}`,
        status: response.status >= 200 && response.status < 300 ? 'success' : 'error',
        duration: 3000,
        isClosable: true,
      })
      
    } catch (error) {
      const errorResponse = {
        error: true,
        message: error.message,
        timestamp: new Date().toISOString(),
        method: editMethod,
        url: editUrl
      }
      
      if (error.response) {
        errorResponse.status = error.response.status
        errorResponse.statusText = error.response.statusText
        errorResponse.data = error.response.data
      }
      
      setLastResponse(errorResponse)
      
      toast({
        title: 'Request Failed',
        description: error.message,
        status: 'error',
        duration: 5000,
        isClosable: true,
      })
    } finally {
      setResponseLoading(false)
    }
  }

  const CodeBlock = ({ children, language = 'bash', title, executable = false, expectedResponse = null }) => {
    const [blockResponse, setBlockResponse] = React.useState(null)
    const [blockLoading, setBlockLoading] = React.useState(false)
    
    const executeBlockRequest = async (codeContent) => {
      try {
        setBlockLoading(true)
        setBlockResponse(null)
        
        // Parse curl command to extract method, url, headers, and body
        const lines = codeContent.split('\n').map(line => line.trim()).filter(line => line)
        const curlLine = lines.find(line => line.startsWith('curl'))
        
        if (!curlLine) {
          toast({
            title: 'Invalid Request',
            description: 'Could not parse curl command from code block',
            status: 'error',
            duration: 3000,
            isClosable: true,
          })
          setBlockLoading(false)
          return
        }

        // Extract method
        const methodMatch = curlLine.match(/-X\s+(\w+)/)
        const method = methodMatch ? methodMatch[1] : 'GET'

        // Extract URL - improved parsing
        let url = '/'
        const quotedUrlMatch = curlLine.match(/"([^"]*)"/)
        if (quotedUrlMatch) {
          url = quotedUrlMatch[1]
        } else {
          // Fallback: look for URL after curl command, avoiding flags
          const parts = curlLine.split(/\s+/)
          for (let i = 1; i < parts.length; i++) {
            const part = parts[i]
            if (!part.startsWith('-') && (part.startsWith('/') || part.startsWith('http'))) {
              url = part
              break
            }
          }
        }
        
        // Replace variables in URL
        url = url.replace(/\$\{serverUrl\}/g, window.location.origin)
        url = url.replace(/\$\{masterKey\}/g, masterKey || 'your_master_key_here')

        // Extract headers
        const headers = {}
        const headerLines = lines.filter(line => line.includes('-H'))
        headerLines.forEach(line => {
          const headerMatch = line.match(/-H\s+"([^:]+):\s*([^"]+)"/)
          if (headerMatch) {
            headers[headerMatch[1]] = headerMatch[2].replace(/\$\{masterKey\}/g, masterKey || 'your_master_key_here')
          }
        })

        // Add authentication headers based on current auth mode
        if (authMode === 'masterkey' && masterKey) {
          headers['X-Master-Key'] = masterKey
        } else if (authMode === 'userpass' && username && password) {
          headers['Authorization'] = `Basic ${btoa(`${username}:${password}`)}`
        }

        // Extract body
        let body = null
        const bodyLines = lines.filter(line => line.includes('-d'))
        if (bodyLines.length > 0) {
          const bodyContent = bodyLines.join(' ').match(/-d\s+'([^']+)'||-d\s+"([^"]+)"/)
          if (bodyContent) {
            body = bodyContent[1] || bodyContent[2]
            try {
              body = JSON.parse(body)
            } catch (e) {
              // Keep as string if not valid JSON
            }
          }
        }

        const startTime = Date.now()
        
        console.log('Executing request:', { method, url, headers, body })
        
        // Execute the request
        const response = await apiService.testEndpoint(method, url, body, headers)
        const endTime = Date.now()
        
        const responseData = {
          status: response.status,
          statusText: response.statusText,
          headers: response.headers && typeof response.headers.entries === 'function' 
            ? Object.fromEntries(response.headers.entries()) 
            : response.headers || {},
          data: response.data,
          duration: endTime - startTime,
          timestamp: new Date().toISOString(),
          method,
          url
        }

        setBlockResponse(responseData)
        
        toast({
          title: 'Request Executed',
          description: `${method} ${url} - ${response.status} ${response.statusText}`,
          status: response.status >= 200 && response.status < 300 ? 'success' : 'error',
          duration: 3000,
          isClosable: true,
        })
        
      } catch (error) {
        const errorResponse = {
          error: true,
          message: error.message,
          timestamp: new Date().toISOString(),
          method: 'Unknown',
          url: 'Unknown'
        }
        
        if (error.response) {
          errorResponse.status = error.response.status
          errorResponse.statusText = error.response.statusText
          errorResponse.data = error.response.data
        }
        
        setBlockResponse(errorResponse)
        
        toast({
          title: 'Request Failed',
          description: error.message,
          status: 'error',
          duration: 5000,
          isClosable: true,
        })
      } finally {
        setBlockLoading(false)
      }
    }

    const openBlockEditor = (codeContent) => {
      // Use the global editor but set a callback to update this block's response
      openRequestEditor(codeContent)
      // We'll need to modify the global editor to support per-block responses
    }

    return (
      <VStack align="stretch" spacing={3} mb={4}>
        <Box position="relative">
          {title && (
            <Text fontSize="sm" fontWeight="medium" mb={2} color="gray.600">
              {title}
            </Text>
          )}
          <Box
            bg={codeBg}
            borderRadius="md"
            border="1px"
            borderColor={useColorModeValue('gray.200', 'gray.600')}
            position="relative"
          >
            <HStack justify="space-between" p={3} borderBottom="1px" borderColor={useColorModeValue('gray.200', 'gray.600')}>
              <Badge colorScheme="blue" variant="subtle">{language}</Badge>
              <HStack>
                {executable && language === 'bash' && children.includes('curl') && (
                  <>
                    <Tooltip label="Edit and execute request">
                      <IconButton
                        size="sm"
                        variant="ghost"
                        icon={<FiEdit />}
                        onClick={() => openBlockEditor(children)}
                        aria-label="Edit request"
                        colorScheme="blue"
                      />
                    </Tooltip>
                    <Tooltip label="Execute request">
                      <IconButton
                        size="sm"
                        variant="ghost"
                        icon={<FiPlay />}
                        onClick={() => executeBlockRequest(children)}
                        aria-label="Execute request"
                        colorScheme="green"
                        isLoading={blockLoading}
                      />
                    </Tooltip>
                  </>
                )}
                <Tooltip label="Copy to clipboard">
                  <IconButton
                    size="sm"
                    variant="ghost"
                    icon={<FiCopy />}
                    onClick={() => copyToClipboard(children)}
                    aria-label="Copy code"
                  />
                </Tooltip>
              </HStack>
            </HStack>
            <Box p={4} fontSize="sm" overflow="auto" maxH="400px" fontFamily="mono" whiteSpace="pre">
              <Code>{children}</Code>
            </Box>
          </Box>
        </Box>

        {/* Response Display for this specific code block */}
        {(blockResponse || expectedResponse) && (
          <Card bg={cardBg} size="sm">
            <CardHeader py={2}>
              <HStack justify="space-between">
                <HStack spacing={2}>
                  <Text fontSize="sm" fontWeight="medium">
                    {blockResponse ? 'API Response' : 'Expected Response'}
                  </Text>
                  {blockResponse?.status && (
                    <Badge size="sm" colorScheme={
                      blockResponse.status >= 200 && blockResponse.status < 300 ? 'green' :
                      blockResponse.status >= 400 && blockResponse.status < 500 ? 'orange' : 'red'
                    } variant="solid">
                      {blockResponse.status}
                    </Badge>
                  )}
                  {blockResponse?.duration && (
                    <Badge size="sm" variant="outline">
                      {blockResponse.duration}ms
                    </Badge>
                  )}
                </HStack>
                <Button size="xs" variant="ghost" onClick={() => setBlockResponse(null)}>
                  ×
                </Button>
              </HStack>
            </CardHeader>
            <CardBody py={2}>
              <Tabs size="sm">
                <TabList>
                  {blockResponse && (
                    <>
                      <Tab fontSize="xs">
                        <HStack spacing={1}>
                          <Text>Actual</Text>
                          <Badge size="xs" colorScheme="green" variant="outline">Live</Badge>
                        </HStack>
                      </Tab>
                      <Tab fontSize="xs">Headers</Tab>
                    </>
                  )}
                  {expectedResponse && (
                    <Tab fontSize="xs">
                      <HStack spacing={1}>
                        <Text>Expected</Text>
                        <Badge size="xs" colorScheme="blue" variant="outline">Doc</Badge>
                      </HStack>
                    </Tab>
                  )}
                </TabList>
                <TabPanels>
                  {blockResponse && (
                    <>
                      <TabPanel p={2}>
                        <Box
                          as="pre"
                          bg={useColorModeValue('gray.50', 'gray.800')}
                          p={3}
                          borderRadius="md"
                          overflow="auto"
                          fontSize="xs"
                          fontFamily="mono"
                          maxH="200px"
                          whiteSpace="pre-wrap"
                        >
                          {blockResponse.error
                            ? blockResponse.message
                            : JSON.stringify(blockResponse.data, null, 2)}
                        </Box>
                      </TabPanel>
                      <TabPanel p={2}>
                        <Box
                          as="pre"
                          bg={useColorModeValue('gray.50', 'gray.800')}
                          p={3}
                          borderRadius="md"
                          overflow="auto"
                          fontSize="xs"
                          fontFamily="mono"
                          maxH="200px"
                        >
                          {JSON.stringify(blockResponse.headers || {}, null, 2)}
                        </Box>
                      </TabPanel>
                    </>
                  )}
                  {expectedResponse && (
                    <TabPanel p={2}>
                      <Box
                        as="pre"
                        bg={useColorModeValue('blue.50', 'blue.900')}
                        p={3}
                        borderRadius="md"
                        overflow="auto"
                        fontSize="xs"
                        fontFamily="mono"
                        maxH="200px"
                        whiteSpace="pre-wrap"
                      >
                        {expectedResponse}
                      </Box>
                    </TabPanel>
                  )}
                </TabPanels>
              </Tabs>
            </CardBody>
          </Card>
        )}
      </VStack>
    )
  }

  const HttpMethodBadge = ({ method }) => {
    const colors = {
      GET: 'green',
      POST: 'blue',
      PUT: 'orange',
      DELETE: 'red',
      PATCH: 'purple'
    }
    return <Badge colorScheme={colors[method] || 'gray'}>{method}</Badge>
  }

  const TableOfContents = ({ sections }) => (
    <Card mb={6}>
      <CardHeader>
        <Heading size="sm">Table of Contents</Heading>
      </CardHeader>
      <CardBody pt={0}>
        <UnorderedList spacing={1}>
          {sections.map((section, index) => (
            <ListItem key={index}>
              <Link href={`#${section.id}`} color="blue.500" fontSize="sm">
                {section.title}
              </Link>
              {section.subsections && (
                <UnorderedList ml={4} mt={1}>
                  {section.subsections.map((sub, subIndex) => (
                    <ListItem key={subIndex}>
                      <Link href={`#${sub.id}`} color="blue.400" fontSize="xs">
                        {sub.title}
                      </Link>
                    </ListItem>
                  ))}
                </UnorderedList>
              )}
            </ListItem>
          ))}
        </UnorderedList>
      </CardBody>
    </Card>
  )

  return (
    <Box position="relative" minH="100vh">
      <AnimatedBackground />
      <Box position="relative" zIndex={1} p={6}>
        <VStack align="stretch" spacing={6}>
      <VStack align="stretch" spacing={4}>
        <HStack justify="space-between">
          <HStack>
            <FiBook />
            <Heading size="lg">API Documentation</Heading>
          </HStack>
          <FormControl maxW="200px">
            <Select
              value={selectedCollection}
              onChange={(e) => setSelectedCollection(e.target.value)}
              placeholder="Select collection"
            >
              {collections.map((collection) => (
                <option key={collection.name} value={collection.name}>
                  {collection.name}
                </option>
              ))}
            </Select>
          </FormControl>
        </HStack>

        <Card bg={cardBg}>
          <CardHeader>
            <HStack spacing={4}>
              <Heading size="md">Test Configuration</Heading>
              <Badge colorScheme="green" variant="outline">
                Live API Testing
              </Badge>
            </HStack>
          </CardHeader>
          <CardBody>
            <VStack align="stretch" spacing={4}>
              <HStack spacing={4}>
                <FormControl>
                  <FormLabel>Authentication Mode</FormLabel>
                  <Select 
                    value={authMode} 
                    onChange={(e) => setAuthMode(e.target.value)}
                    maxW="200px"
                  >
                    <option value="masterkey">Master Key</option>
                    <option value="userpass">Username & Password</option>
                  </Select>
                </FormControl>
                
                <FormControl>
                  <FormLabel>Server URL</FormLabel>
                  <Input
                    value={serverUrl}
                    onChange={(e) => setServerUrl(e.target.value)}
                    placeholder="http://localhost:2403"
                    isReadOnly
                    bg={useColorModeValue('gray.50', 'gray.600')}
                  />
                </FormControl>
              </HStack>

              {authMode === 'masterkey' ? (
                <FormControl>
                  <FormLabel>Master Key</FormLabel>
                  <Input
                    type="password"
                    value={masterKey}
                    onChange={(e) => setMasterKey(e.target.value)}
                    placeholder="mk_your_master_key_here"
                  />
                </FormControl>
              ) : (
                <HStack spacing={4}>
                  <FormControl>
                    <FormLabel>Username</FormLabel>
                    <Input
                      value={username}
                      onChange={(e) => setUsername(e.target.value)}
                      placeholder="username@example.com"
                    />
                  </FormControl>
                  <FormControl>
                    <FormLabel>Password</FormLabel>
                    <Input
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      placeholder="password"
                    />
                  </FormControl>
                </HStack>
              )}
              
              <Text fontSize="sm" color="gray.500">
                💡 Configure your credentials above, then click the ▶️ play buttons on curl examples to test them live!
              </Text>
            </VStack>
          </CardBody>
        </Card>
      </VStack>


      <Tabs>
        <TabList>
          <Tab><HStack><FiDatabase /><Text>Collections API</Text></HStack></Tab>
          <Tab><HStack><FiKey /><Text>Master Key Auth</Text></HStack></Tab>
          <Tab><HStack><FiUsers /><Text>User Management</Text></HStack></Tab>
          <Tab><HStack><FiShield /><Text>Authentication</Text></HStack></Tab>
          <Tab><HStack><FiServer /><Text>Admin API</Text></HStack></Tab>
          <Tab><HStack><FiCode /><Text>Events System</Text></HStack></Tab>
          <Tab><HStack><FiDatabase /><Text>Database Config</Text></HStack></Tab>
          <Tab><HStack><FiServer /><Text>WebSocket & Real-time</Text></HStack></Tab>
          <Tab><HStack><FiCode /><Text>dpd.js Client</Text></HStack></Tab>
          <Tab><HStack><FiDatabase /><Text>Advanced Queries</Text></HStack></Tab>
        </TabList>

        <TabPanels>
          {/* Collections API Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <TableOfContents sections={[
                {
                  id: 'crud-operations',
                  title: 'Basic CRUD Operations',
                  subsections: [
                    { id: 'get-all', title: 'GET All Documents' },
                    { id: 'get-single', title: 'GET Single Document' },
                    { id: 'post-create', title: 'POST Create Document' },
                    { id: 'put-update', title: 'PUT Update Document' },
                    { id: 'delete-doc', title: 'DELETE Document' }
                  ]
                },
                {
                  id: 'advanced-queries',
                  title: 'Advanced Queries',
                  subsections: [
                    { id: 'filtering', title: 'Filtering' },
                    { id: 'mongodb-operators', title: 'MongoDB-Style Operators' },
                    { id: 'sorting-pagination', title: 'Sorting & Pagination' }
                  ]
                }
              ]} />


              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Basic CRUD Operations - {selectedCollection}</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={6}>
                    
                    {/* GET All Documents */}
                    <Box>
                      <HStack mb={3}>
                        <HttpMethodBadge method="GET" />
                        <Text fontWeight="bold">Get All Documents</Text>
                      </HStack>
                      <CodeBlock 
                        language="bash" 
                        title="Get all documents" 
                        executable
                        expectedResponse={`[
  {
    "id": "doc123",
    "title": "Example Document",
    "createdAt": "2024-06-22T10:00:00Z",
    "updatedAt": "2024-06-22T10:00:00Z"
  }
]`}
                      >
{`curl -X GET "${serverUrl}/${selectedCollection}"`}
                      </CodeBlock>
                    </Box>

                    {/* GET Single Document */}
                    <Box>
                      <HStack mb={3}>
                        <HttpMethodBadge method="GET" />
                        <Text fontWeight="bold">Get Single Document</Text>
                      </HStack>
                      <CodeBlock language="bash" title="Get document by ID" executable>
{`curl -X GET "${serverUrl}/${selectedCollection}/doc123"`}
                      </CodeBlock>
                    </Box>

                    {/* POST Create Document */}
                    <Box>
                      <HStack mb={3}>
                        <HttpMethodBadge method="POST" />
                        <Text fontWeight="bold">Create Document</Text>
                      </HStack>
                      <CodeBlock language="bash" title="Create new document" executable>
{`curl -X POST "${serverUrl}/${selectedCollection}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "title": "New Document",
    "content": "Document content here",
    "tags": ["example", "api"]
  }'`}
                      </CodeBlock>
                    </Box>

                    {/* PUT Update Document */}
                    <Box>
                      <HStack mb={3}>
                        <HttpMethodBadge method="PUT" />
                        <Text fontWeight="bold">Update Document</Text>
                      </HStack>
                      <CodeBlock language="bash" title="Update existing document">
{`curl -X PUT "${serverUrl}/${selectedCollection}/doc123" \\
  -H "Content-Type: application/json" \\
  -d '{
    "title": "Updated Document",
    "content": "Updated content"
  }'`}
                      </CodeBlock>
                    </Box>

                    {/* DELETE Document */}
                    <Box>
                      <HStack mb={3}>
                        <HttpMethodBadge method="DELETE" />
                        <Text fontWeight="bold">Delete Document</Text>
                      </HStack>
                      <CodeBlock language="bash" title="Delete document">
{`curl -X DELETE "${serverUrl}/${selectedCollection}/doc123"`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Advanced Queries</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={6}>
                    
                    {/* Filtering */}
                    <Box>
                      <Text fontWeight="bold" mb={3}>Filtering</Text>
                      <CodeBlock language="bash" title="Simple filtering">
{`# Filter by field value
curl "${serverUrl}/${selectedCollection}?status=active"

# Multiple filters
curl "${serverUrl}/${selectedCollection}?status=active&priority=high"`}
                      </CodeBlock>
                    </Box>

                    {/* MongoDB Operators */}
                    <Box>
                      <Text fontWeight="bold" mb={3}>MongoDB-Style Operators</Text>
                      <CodeBlock language="bash" title="Advanced filtering with operators">
{`# Greater than / Less than
curl "${serverUrl}/${selectedCollection}?age={\\"\\$gt\\":18}"
curl "${serverUrl}/${selectedCollection}?price={\\"\\$lte\\":100}"

# In array
curl "${serverUrl}/${selectedCollection}?status={\\"\\$in\\":[\\"active\\",\\"pending\\"]}"

# Not equal
curl "${serverUrl}/${selectedCollection}?status={\\"\\$ne\\":\\"deleted\\"}"

# Exists
curl "${serverUrl}/${selectedCollection}?email={\\"\\$exists\\":true}"`}
                      </CodeBlock>
                    </Box>

                    {/* Sorting and Pagination */}
                    <Box>
                      <Text fontWeight="bold" mb={3}>Sorting & Pagination</Text>
                      <CodeBlock language="bash" title="Sorting and pagination">
{`# Sort ascending
curl "${serverUrl}/${selectedCollection}?\\$sort={\\"createdAt\\":1}"

# Sort descending
curl "${serverUrl}/${selectedCollection}?\\$sort={\\"createdAt\\":-1}"

# Limit results
curl "${serverUrl}/${selectedCollection}?\\$limit=10"

# Skip results (pagination)
curl "${serverUrl}/${selectedCollection}?\\$skip=20&\\$limit=10"

# Select specific fields
curl "${serverUrl}/${selectedCollection}?\\$fields={\\"title\\":1,\\"status\\":1}"`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Master Key Auth Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <TableOfContents sections={[
                {
                  id: 'master-key-overview',
                  title: 'Master Key Overview',
                  subsections: [
                    { id: 'key-features', title: 'Key Features' },
                    { id: 'configuration', title: 'Configuration Location' }
                  ]
                },
                {
                  id: 'api-usage',
                  title: 'Using Master Key in API Calls',
                  subsections: [
                    { id: 'via-header', title: 'Via Header (Recommended)' },
                    { id: 'via-auth-header', title: 'Via Authorization Header' },
                    { id: 'dashboard-login', title: 'Dashboard Login' },
                    { id: 'system-login', title: 'System Login (Programmatic)' }
                  ]
                }
              ]} />


              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Master Key Overview</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <Text>
                      The master key is auto-generated on first startup and stored securely in <Code>.deployd/security.json</Code> with 
                      600 permissions (owner read/write only).
                    </Text>
                    
                    <Box>
                      <Text fontWeight="bold" mb={2}>Key Features:</Text>
                      <VStack align="start" spacing={1}>
                        <HStack><Badge colorScheme="green">✓</Badge><Text fontSize="sm">96-character cryptographically secure key</Text></HStack>
                        <HStack><Badge colorScheme="green">✓</Badge><Text fontSize="sm">Dashboard authentication</Text></HStack>
                        <HStack><Badge colorScheme="green">✓</Badge><Text fontSize="sm">Admin API protection</Text></HStack>
                        <HStack><Badge colorScheme="green">✓</Badge><Text fontSize="sm">User management capabilities</Text></HStack>
                        <HStack><Badge colorScheme="green">✓</Badge><Text fontSize="sm">isRoot=true privileges</Text></HStack>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Configuration Location:</Text>
                      <CodeBlock language="json" title=".deployd/security.json">
{`{
  "masterKey": "mk_...",
  "jwtSecret": "your-jwt-secret-key",
  "jwtExpiration": "24h",
  "allowRegistration": false
}`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Using Master Key in API Calls</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={6}>
                    
                    <Box>
                      <Text fontWeight="bold" mb={3}>Via Header (Recommended)</Text>
                      <CodeBlock language="bash" title="X-Master-Key header" executable>
{`curl -H "X-Master-Key: ${masterKey}" \\
  "${serverUrl}/_admin/info"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Via Authorization Header</Text>
                      <CodeBlock language="bash" title="Bearer token format">
{`curl -H "Authorization: Bearer ${masterKey}" \\
  "${serverUrl}/_admin/info"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Dashboard Login</Text>
                      <CodeBlock language="bash" title="Dashboard authentication">
{`curl -X POST "${serverUrl}/_admin/auth/dashboard-login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "masterKey": "${masterKey}"
  }'`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>System Login (Programmatic)</Text>
                      <CodeBlock language="bash" title="System login for SSO integration">
{`curl -X POST "${serverUrl}/_admin/auth/system-login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "username": "admin@example.com",
    "masterKey": "${masterKey}"
  }'`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* User Management Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <TableOfContents sections={[
                {
                  id: 'create-user',
                  title: 'Create User (Master Key Required)'
                },
                {
                  id: 'user-auth',
                  title: 'User Authentication',
                  subsections: [
                    { id: 'standard-login', title: 'Standard Login' },
                    { id: 'get-current-user', title: 'Get Current User (/me)' },
                    { id: 'logout', title: 'Logout' }
                  ]
                },
                {
                  id: 'user-roles',
                  title: 'User Roles & Permissions'
                }
              ]} />


              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Create User (Master Key Required)</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <CodeBlock language="bash" title="Create new user">
{`curl -X POST "${serverUrl}/_admin/auth/create-user" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: ${masterKey}" \\
  -d '{
    "masterKey": "${masterKey}",
    "userData": {
      "username": "newuser",
      "email": "user@example.com",
      "password": "securepassword",
      "role": "user"
    }
  }'`}
                    </CodeBlock>
                    
                    <Text fontSize="sm" color="gray.600">Response:</Text>
                    <CodeBlock language="json">
{`{
  "success": true,
  "message": "User created successfully",
  "user": {
    "id": "user123",
    "username": "newuser",
    "email": "user@example.com",
    "role": "user",
    "createdAt": "2024-06-22T10:00:00Z"
  }
}`}
                    </CodeBlock>
                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">User Authentication</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={6}>
                    
                    <Box>
                      <Text fontWeight="bold" mb={3}>JWT Login with Master Key</Text>
                      <Text mb={2} fontSize="sm" color="green.600">
                        ✅ Master key authentication returns a JWT token with root privileges.
                      </Text>
                      <CodeBlock language="bash" title="Login with master key to get JWT" executable>
{`curl -X POST "${serverUrl}/auth/login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "masterKey": "${masterKey}"
  }'`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>JWT Login with Username/Password</Text>
                      <Text mb={2} fontSize="sm" color="green.600">
                        ✅ User/password authentication is fully supported and returns a JWT token.
                      </Text>
                      <CodeBlock language="bash" title="Login with username/password to get JWT" executable>
{`curl -X POST "${serverUrl}/auth/login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "username": "john@example.com",
    "password": "securePassword123"
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>Response format:</Text>
                      <CodeBlock language="json">
{`{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expiresAt": 1719489600,
  "isRoot": false
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Get Current User (/auth/me)</Text>
                      <Text mb={2}>Use the JWT token to get user information:</Text>
                      <CodeBlock language="bash" title="Get current user info with JWT" executable>
{`# Using JWT token from login response
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \\
  "${serverUrl}/auth/me"`}
                      </CodeBlock>
                    </Box>

                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">User Roles & Permissions</Heading>
                </CardHeader>
                <CardBody>
                  <Table variant="simple">
                    <Thead>
                      <Tr>
                        <Th>Role</Th>
                        <Th>Permissions</Th>
                        <Th>isRoot</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      <Tr>
                        <Td><Badge>user</Badge></Td>
                        <Td>Read/write own documents, basic API access</Td>
                        <Td><Badge colorScheme="red">false</Badge></Td>
                      </Tr>
                      <Tr>
                        <Td><Badge colorScheme="orange">admin</Badge></Td>
                        <Td>Read/write all documents, user management</Td>
                        <Td><Badge colorScheme="green">true</Badge></Td>
                      </Tr>
                      <Tr>
                        <Td><Badge colorScheme="purple">master key</Badge></Td>
                        <Td>Full system access, all admin operations</Td>
                        <Td><Badge colorScheme="green">true</Badge></Td>
                      </Tr>
                    </Tbody>
                  </Table>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Authentication Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <TableOfContents sections={[
                {
                  id: 'jwt-auth-flow',
                  title: 'JWT Authentication Flow',
                  subsections: [
                    { id: 'jwt-overview', title: 'Overview' },
                    { id: 'create-user-jwt', title: 'Step 1: Create User' },
                    { id: 'login-jwt', title: 'Step 2: Login and Get Token' },
                    { id: 'use-token', title: 'Step 3: Use Token for API Calls' },
                    { id: 'get-user-info', title: 'Step 4: Get User Info with /auth/me' }
                  ]
                },
                {
                  id: 'security-features',
                  title: 'Security Features'
                },
              ]} />

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">JWT Authentication Flow</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={6}>
                    <Box>
                      <Text fontWeight="bold" mb={3} id="jwt-overview">Overview</Text>
                      <Text>
                        Go-Deployd uses JWT (JSON Web Tokens) for stateless authentication. This complete example 
                        shows how to create a user, login to get a JWT token, and use that token to access protected endpoints.
                      </Text>
                      <Text mt={2} fontSize="sm" color="green.600">
                        ✅ Both master key and user/password authentication are fully supported with JWT tokens.
                        You can authenticate with either a master key (for admin operations) or username/password (for user operations).
                      </Text>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3} id="create-user-jwt">Step 1: Create User (Master Key Required)</Text>
                      <Text mb={2}>First, create a user using the master key:</Text>
                      <CodeBlock language="bash" title="Create a new user" executable>
{`curl -X POST "${serverUrl}/_admin/auth/create-user" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: ${masterKey}" \\
  -d '{
    "userData": {
      "username": "johndoe",
      "email": "john@example.com",
      "password": "securePassword123",
      "name": "John Doe",
      "role": "user"
    }
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>Response:</Text>
                      <CodeBlock language="json">
{`{
  "success": true,
  "message": "User created successfully",
  "user": {
    "id": "65f7a8b9c1234567890abcde",
    "username": "johndoe",
    "email": "john@example.com",
    "name": "John Doe",
    "role": "user",
    "createdAt": "2024-06-26T10:00:00Z"
  }
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3} id="login-jwt">Step 2: Login and Get JWT Token</Text>
                      <Text mb={2}>Login with either master key OR username/password to get a JWT token:</Text>
                      
                      <Text fontWeight="medium" mb={2} color="blue.600">Option A: Master Key Login (Root Privileges)</Text>
                      <CodeBlock language="bash" title="Login with master key" executable>
{`curl -X POST "${serverUrl}/auth/login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "masterKey": "${masterKey}"
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>Response:</Text>
                      <CodeBlock language="json">
{`{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiJyb290IiwidXNlcm5hbWUiOiJyb290IiwiaXNSb290Ijp0cnVlLCJleHAiOjE3MTk0ODk2MDB9.xyzabc123...",
  "expiresAt": 1719489600,
  "isRoot": true
}`}
                      </CodeBlock>
                      
                      <Text fontWeight="medium" mb={2} mt={4} color="green.600">Option B: Username/Password Login (User Privileges)</Text>
                      <CodeBlock language="bash" title="Login with username/password" executable>
{`curl -X POST "${serverUrl}/auth/login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "username": "johndoe",
    "password": "securePassword123"
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>Response:</Text>
                      <CodeBlock language="json">
{`{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VySWQiOiI2NWY3YThiOWMxMjM0NTY3ODkwYWJjZGUiLCJ1c2VybmFtZSI6ImpvaG5kb2UiLCJpc1Jvb3QiOmZhbHNlLCJleHAiOjE3MTk0ODk2MDB9.abc123xyz...",
  "expiresAt": 1719489600,
  "isRoot": false
}`}
                      </CodeBlock>
                      <Text fontSize="sm" color="green.600" mt={2}>
                        ✅ Both master key and user/password authentication are fully supported!
                      </Text>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3} id="use-token">Step 3: Use Token for API Calls</Text>
                      <Text mb={2}>Use the JWT token in the Authorization header for all subsequent API calls:</Text>
                      <CodeBlock language="bash" title="Save token to variable">
{`# Save the token from the login response
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`}
                      </CodeBlock>
                      <CodeBlock language="bash" title="Use token in API calls" executable>
{`# Get all users
curl -H "Authorization: Bearer $TOKEN" \\
  "${serverUrl}/users"

# Create a document
curl -X POST "${serverUrl}/todos" \\
  -H "Authorization: Bearer $TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{
    "title": "Complete JWT implementation",
    "completed": false
  }'

# Access admin endpoints
curl -H "Authorization: Bearer $TOKEN" \\
  "${serverUrl}/_admin/collections"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3} id="get-user-info">Step 4: Get User Info with /auth/me</Text>
                      <Text mb={2}>Use the `/auth/me` endpoint to get information about the currently authenticated user:</Text>
                      <CodeBlock language="bash" title="Get current user info" executable>
{`curl -H "Authorization: Bearer $TOKEN" \\
  "${serverUrl}/auth/me"`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>Response for root user:</Text>
                      <CodeBlock language="json">
{`{
  "id": "root",
  "username": "root",
  "isRoot": true
}`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>Response for regular user (when implemented):</Text>
                      <CodeBlock language="json">
{`{
  "id": "65f7a8b9c1234567890abcde",
  "username": "johndoe",
  "email": "john@example.com",
  "name": "John Doe",
  "role": "user",
  "createdAt": "2024-06-26T10:00:00Z",
  "updatedAt": "2024-06-26T10:00:00Z"
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Complete Example Script</Text>
                      <Text mb={2}>Here's a complete bash script demonstrating the full flow:</Text>
                      <CodeBlock language="bash" title="complete-jwt-flow.sh">
{`#!/bin/bash

# Configuration
SERVER_URL="http://localhost:2403"
MASTER_KEY="mk_your_master_key_here"

echo "1. Creating user..."
CREATE_RESPONSE=$(curl -s -X POST "$SERVER_URL/_admin/auth/create-user" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: $MASTER_KEY" \\
  -d '{
    "userData": {
      "username": "testuser",
      "email": "test@example.com",
      "password": "testPassword123",
      "name": "Test User"
    }
  }')

echo "User created: $(echo $CREATE_RESPONSE | jq -r '.user.username')"

echo -e "\\n2. Logging in with master key..."
LOGIN_RESPONSE=$(curl -s -X POST "$SERVER_URL/auth/login" \\
  -H "Content-Type: application/json" \\
  -d "{\"masterKey\": \"$MASTER_KEY\"}")

TOKEN=$(echo $LOGIN_RESPONSE | jq -r '.token')
echo "Got JWT token (master key): \${TOKEN:0:50}..."

echo -e "\\n2b. Alternative: Login with username/password..."
USER_LOGIN_RESPONSE=$(curl -s -X POST "$SERVER_URL/auth/login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "username": "testuser",
    "password": "testPassword123"
  }')

USER_TOKEN=$(echo $USER_LOGIN_RESPONSE | jq -r '.token')
echo "Got JWT token (user): \${USER_TOKEN:0:50}..."
echo "Using master key token for admin operations..."

echo -e "\\n3. Using token to create a todo..."
TODO_RESPONSE=$(curl -s -X POST "$SERVER_URL/todos" \\
  -H "Authorization: Bearer $TOKEN" \\
  -H "Content-Type: application/json" \\
  -d '{
    "title": "Test todo from JWT",
    "completed": false
  }')

echo "Created todo: $(echo $TODO_RESPONSE | jq -r '.title')"

echo -e "\\n4. Getting user info with /auth/me..."
ME_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \\
  "$SERVER_URL/auth/me")

echo "Current user: $(echo $ME_RESPONSE | jq '.')"

echo -e "\\n5. Validating token..."
VALIDATE_RESPONSE=$(curl -s -H "Authorization: Bearer $TOKEN" \\
  "$SERVER_URL/auth/validate")

echo "Token validation: $(echo $VALIDATE_RESPONSE | jq '.')"
`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>JWT Token Structure</Text>
                      <Text mb={2}>The JWT tokens contain the following claims:</Text>
                      <CodeBlock language="json">
{`{
  "userId": "root",           // User ID (or "root" for master key)
  "username": "root",         // Username
  "isRoot": true,            // Whether user has root privileges
  "exp": 1719489600,         // Expiration timestamp
  "iat": 1719403200          // Issued at timestamp
}`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>
                        • Tokens expire after 24 hours by default (configurable)
                        <br />
                        • Only minimal user data is stored in the token
                        <br />
                        • Full user data is fetched from the database when needed
                      </Text>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Security Features</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="start" spacing={3}>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>bcrypt password hashing (cost 12)</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>JWT token authentication with secure signing</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>Master key authentication (96-char secure key)</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>File permissions (600) for sensitive config</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>Role-based access control (RBAC)</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>Document-level access filtering</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>CORS protection</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>Input validation and sanitization</Text></HStack>
                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">JWT Token Management</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <Text>
                      Go-Deployd uses JWT (JSON Web Tokens) for stateless authentication. Tokens are validated 
                      on each request without requiring server-side session storage.
                    </Text>
                    
                    <Box>
                      <Text fontWeight="bold" mb={2}>JWT Token Properties:</Text>
                      <VStack align="start" spacing={1}>
                        <Text fontSize="sm">• <strong>Expiration:</strong> 24 hours (configurable via JWTExpiration setting)</Text>
                        <Text fontSize="sm">• <strong>Storage:</strong> Client-side (localStorage, cookies, or environment variables)</Text>
                        <Text fontSize="sm">• <strong>Security:</strong> HMAC-SHA256 signed with secret key</Text>
                        <Text fontSize="sm">• <strong>Claims:</strong> User ID, username, isRoot flag, expiration time</Text>
                        <Text fontSize="sm">• <strong>Stateless:</strong> No server-side storage required</Text>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Token Validation:</Text>
                      <CodeBlock language="bash" title="Validate JWT token" executable>
{`# Validate current token
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \\
  "${serverUrl}/auth/validate"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Using Tokens in Requests:</Text>
                      <CodeBlock language="bash" title="JWT token usage patterns">
{`# Standard Bearer token (recommended)
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \\
  "${serverUrl}/api/endpoint"

# Alternative: Store in environment variable
export JWT_TOKEN="your_jwt_token_here"
curl -H "Authorization: Bearer $JWT_TOKEN" \\
  "${serverUrl}/api/endpoint"

# For CLI tools: Save to file
echo "your_jwt_token" > ~/.deployd-token
curl -H "Authorization: Bearer $(cat ~/.deployd-token)" \\
  "${serverUrl}/api/endpoint"`}
                      </CodeBlock>
                    </Box>

                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Admin API Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <TableOfContents sections={[
                {
                  id: 'server-info',
                  title: 'Server Information'
                },
                {
                  id: 'collection-mgmt',
                  title: 'Collection Management',
                  subsections: [
                    { id: 'list-collections', title: 'List Collections' },
                    { id: 'get-collection-details', title: 'Get Collection Details' },
                    { id: 'create-collection', title: 'Create Collection' }
                  ]
                },
                {
                  id: 'security-settings',
                  title: 'Security Settings Management',
                  subsections: [
                    { id: 'get-security-settings', title: 'Get Security Settings' },
                    { id: 'update-security-settings', title: 'Update Security Settings' },
                    { id: 'validate-master-key', title: 'Validate Master Key' }
                  ]
                }
              ]} />


              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Server Information</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <CodeBlock language="bash" title="Get server info">
{`curl -H "X-Master-Key: ${masterKey}" \\
  "${serverUrl}/_admin/info"`}
                    </CodeBlock>
                    
                    <Text fontSize="sm" color="gray.600">Response:</Text>
                    <CodeBlock language="json">
{`{
  "version": "1.0.0",
  "goVersion": "go1.21",
  "uptime": "2h 15m",
  "database": "Connected",
  "environment": "development"
}`}
                    </CodeBlock>
                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Collection Management</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={6}>
                    
                    <Box>
                      <Text fontWeight="bold" mb={3}>List Collections</Text>
                      <CodeBlock language="bash">
{`curl -H "X-Master-Key: ${masterKey}" \\
  "${serverUrl}/_admin/collections"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Get Collection Details</Text>
                      <CodeBlock language="bash">
{`curl -H "X-Master-Key: ${masterKey}" \\
  "${serverUrl}/_admin/collections/users"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Create Collection</Text>
                      <CodeBlock language="bash">
{`curl -X POST "${serverUrl}/_admin/collections/products" \\
  -H "X-Master-Key: ${masterKey}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": {"type": "string", "required": true},
    "price": {"type": "number", "required": true},
    "category": {"type": "string", "default": "general"}
  }'`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Security Settings Management</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={6}>
                    
                    <Box>
                      <Text fontWeight="bold" mb={3}>Get Security Settings</Text>
                      <CodeBlock language="bash">
{`curl -H "X-Master-Key: ${masterKey}" \\
  "${serverUrl}/_admin/settings/security"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Update Security Settings</Text>
                      <CodeBlock language="bash">
{`curl -X PUT "${serverUrl}/_admin/settings/security" \\
  -H "X-Master-Key: ${masterKey}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "jwtExpiration": "24h",
    "allowRegistration": false
  }'`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Validate Master Key</Text>
                      <CodeBlock language="bash">
{`curl -X POST "${serverUrl}/_admin/auth/validate-master-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "masterKey": "${masterKey}"
  }'`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>
          <TabPanel>
            <VStack spacing={6} align="stretch">
              <TableOfContents sections={[
                {
                  id: 'events-overview',
                  title: 'Events System Overview',
                  subsections: [
                    { id: 'event-types', title: 'Event Types' },
                    { id: 'event-context', title: 'Event Context' }
                  ]
                },
                {
                  id: 'bypass-events',
                  title: 'Bypassing Events (Admin Only)',
                  subsections: [
                    { id: 'skip-in-body', title: 'Using $skipEvents in Request Body' },
                    { id: 'skip-in-query', title: 'Using $skipEvents as Query Parameter' },
                    { id: 'security-notes', title: 'Security Notes' }
                  ]
                },
                {
                  id: 'javascript-events',
                  title: 'JavaScript Events',
                  subsections: [
                    { id: 'js-validation', title: 'Basic Validation Example' },
                    { id: 'npm-modules', title: 'Using npm Modules' },
                    { id: 'js-logging', title: 'Logging and Debugging' },
                    { id: 'js-globals', title: 'Available Global Functions' }
                  ]
                },
                {
                  id: 'go-events',
                  title: 'Go Events',
                  subsections: [
                    { id: 'go-validation', title: 'Basic Validation Example' },
                    { id: 'go-packages', title: 'Using Third-Party Packages' },
                    { id: 'go-logging', title: 'Logging and Debugging' },
                    { id: 'go-methods', title: 'Available EventContext Methods' }
                  ]
                }
              ]} />

              <Card>
                <CardHeader>
                  <Heading size="md">Events System Overview</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      Go-Deployd supports both JavaScript and Go events that run during collection operations. 
                      Events allow you to validate data, modify requests, and implement business logic.
                    </Text>
                    
                    <Box>
                      <Text fontWeight="bold" mb={2}>Event Types</Text>
                      <VStack spacing={2} align="stretch">
                        <Text>• <Code>validate.js/go</Code> - Runs before data is saved (POST/PUT)</Text>
                        <Text>• <Code>post.js/go</Code> - Runs after successful POST operations</Text>
                        <Text>• <Code>put.js/go</Code> - Runs after successful PUT operations</Text>
                        <Text>• <Code>get.js/go</Code> - Runs during GET operations to modify response</Text>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Event Context</Text>
                      <Text>All events receive a context object with:</Text>
                      <VStack spacing={1} align="stretch" mt={2}>
                        <Text>• <Code>data</Code> - The request/response data</Text>
                        <Text>• <Code>query</Code> - URL query parameters</Text>
                        <Text>• <Code>me</Code> - Current authenticated user (if any)</Text>
                        <Text>• <Code>isRoot</Code> - True if user has root privileges</Text>
                        <Text>• <Code>previous</Code> - Previous data (PUT events only)</Text>
                      </VStack>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">Bypassing Events (Admin Only)</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      When using the master key for administrative operations, you can bypass all events 
                      using the special <Code>$skipEvents</Code> parameter. This is useful for data 
                      migrations, bulk operations, or emergency fixes.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Using $skipEvents in Request Body</Text>
                      <CodeBlock language="javascript">
{`// POST/PUT request with $skipEvents in payload
var payload = {
  userId: user_id,
  title: "Admin Created Item",
  $skipEvents: true
};

fetch("/api/collection", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "Authorization": "Bearer " + masterKey
  },
  body: JSON.stringify(payload)
});`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Using $skipEvents as Query Parameter</Text>
                      <CodeBlock language="bash">
{`# GET request bypassing events
curl -X GET "http://localhost:2403/users?$skipEvents=true" \\
  -H "Authorization: Bearer \${MASTER_KEY}"

# POST request bypassing events  
curl -X POST "http://localhost:2403/users?$skipEvents=true" \\
  -H "Authorization: Bearer \${MASTER_KEY}" \\
  -H "Content-Type: application/json" \\
  -d '{"username": "admin", "role": "administrator"}'`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Security Notes</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>⚠️ Only works with valid master key authentication</Text>
                        <Text>⚠️ Bypasses ALL events (validate, post, put, get)</Text>
                        <Text>⚠️ Use carefully - no validation or business logic will run</Text>
                        <Text>✅ Ideal for administrative data operations and migrations</Text>
                      </VStack>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">JavaScript Events</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      JavaScript events run in a V8 engine with support for npm modules and ES6+ features.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Basic Validation Example</Text>
                      <CodeBlock language="javascript">
{`// validate.js
if (!this.title || this.title.length < 3) {
  error('title', 'Title must be at least 3 characters');
}

if (this.email && !/^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$/.test(this.email)) {
  error('email', 'Please enter a valid email address');
}

// Hide sensitive fields
hide('password');`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Using npm Modules</Text>
                      <CodeBlock language="javascript">
{`// post.js - Using external libraries
const bcrypt = require('bcrypt');
const uuid = require('uuid');

if (this.password) {
  // Hash password before saving
  this.password = bcrypt.hashSync(this.password, 10);
}

// Add unique ID
this.externalId = uuid.v4();

// Send welcome email (example)
const nodemailer = require('nodemailer');
// ... email setup and sending logic`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Logging and Debugging</Text>
                      <Text mb={2}>
                        JavaScript events have access to <Code>deployd.log()</Code> for structured logging that 
                        integrates with the server's logging system.
                      </Text>
                      <CodeBlock language="javascript">
{`// Basic logging
deployd.log("User action performed");

// Structured logging with data
deployd.log("Pet created", {
    name: data.name,
    species: data.species,
    user: me,
    timestamp: new Date()
});

// Conditional logging
if (me && me.role === 'admin') {
    deployd.log("Admin action", {
        action: "bulk_update",
        affectedDocs: updateCount,
        adminUser: me.username
    });
}`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>
                        💡 Logging is automatically disabled in production mode for performance. 
                        Logs appear in server output with source identification (e.g., "js:todos").
                      </Text>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Available Global Functions</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>• <Code>deployd.log(message, data)</Code> - Structured logging (development only)</Text>
                        <Text>• <Code>error(field, message)</Code> - Add validation error</Text>
                        <Text>• <Code>hide(field)</Code> - Remove field from response</Text>
                        <Text>• <Code>protect(field)</Code> - Remove field from data</Text>
                        <Text>• <Code>cancel(message, statusCode)</Code> - Cancel operation</Text>
                        <Text>• <Code>isMe(userId)</Code> - Check if user owns resource</Text>
                      </VStack>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">Go Events</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      Go events are compiled as plugins and offer better performance for complex logic.
                      They support any Go module available on the Go module proxy.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Basic Validation Example</Text>
                      <CodeBlock language="go">
{`// validate.go
package main

import (
    "strings"
    "regexp"
)

type EventHandler struct{}

func (h *EventHandler) Run(ctx interface{}) error {
    eventCtx := ctx.(*EventContext)
    
    // Validate title
    if title, ok := eventCtx.Data["title"].(string); !ok || len(title) < 3 {
        eventCtx.Error("title", "Title must be at least 3 characters")
    }
    
    // Validate email format
    if email, ok := eventCtx.Data["email"].(string); ok && email != "" {
        emailRegex := regexp.MustCompile(\`^[^\\s@]+@[^\\s@]+\\.[^\\s@]+$\`)
        if !emailRegex.MatchString(email) {
            eventCtx.Error("email", "Please enter a valid email address")
        }
    }
    
    // Hide sensitive field
    eventCtx.Hide("password")
    
    return nil
}

var EventHandler = &EventHandler{}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Using Third-Party Packages</Text>
                      <CodeBlock language="go">
{`// post.go - Using external libraries
package main

import (
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "golang.org/x/crypto/bcrypt"
)

type EventHandler struct{}

func (h *EventHandler) Run(ctx interface{}) error {
    eventCtx := ctx.(*EventContext)
    
    // Hash password if provided
    if password, ok := eventCtx.Data["password"].(string); ok && password != "" {
        hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
        if err != nil {
            eventCtx.Error("password", "Failed to process password")
            return nil
        }
        eventCtx.Data["password"] = string(hashed)
    }
    
    // Add unique external ID
    eventCtx.Data["externalId"] = uuid.New().String()
    
    // Handle decimal calculations
    if priceStr, ok := eventCtx.Data["price"].(string); ok {
        price, err := decimal.NewFromString(priceStr)
        if err == nil {
            tax := price.Mul(decimal.NewFromFloat(0.1)) // 10% tax
            eventCtx.Data["taxAmount"] = tax.String()
            eventCtx.Data["totalPrice"] = price.Add(tax).String()
        }
    }
    
    return nil
}

var EventHandler = &EventHandler{}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Logging and Debugging</Text>
                      <Text mb={2}>
                        Go events have access to <Code>deployd.Log()</Code> for structured logging that 
                        integrates with the server's logging system.
                      </Text>
                      <CodeBlock language="go">
{`// Basic logging
deployd.Log("User action performed")

// Structured logging with data
deployd.Log("Pet created", map[string]interface{}{
    "name":      ctx.Data["name"],
    "species":   ctx.Data["species"],
    "user":      ctx.Me,
    "timestamp": time.Now(),
})

// Conditional logging
if ctx.IsRoot {
    deployd.Log("Admin action", map[string]interface{}{
        "action":       "bulk_update",
        "affectedDocs": updateCount,
        "adminUser":    ctx.Me["username"],
    })
}`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mt={2}>
                        💡 Logging is automatically disabled in production mode for performance. 
                        Logs appear in server output with source identification (e.g., "go:todos").
                      </Text>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Available EventContext Methods</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>• <Code>deployd.Log(message, data)</Code> - Structured logging (development only)</Text>
                        <Text>• <Code>Error(field, message)</Code> - Add validation error</Text>
                        <Text>• <Code>Hide(field)</Code> - Remove field from response</Text>
                        <Text>• <Code>Protect(field)</Code> - Remove field from data</Text>
                        <Text>• <Code>Cancel(message, statusCode)</Code> - Cancel operation</Text>
                        <Text>• <Code>IsMe(userId)</Code> - Check if user owns resource</Text>
                        <Text>• <Code>HasErrors()</Code> - Check if validation errors exist</Text>
                      </VStack>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          <TabPanel>
            <VStack spacing={6} align="stretch">
              <TableOfContents sections={[
                {
                  id: 'database-config',
                  title: 'Database Configuration',
                  subsections: [
                    { id: 'mongodb-config', title: 'MongoDB Configuration' },
                    { id: 'sqlite-config', title: 'SQLite Configuration' },
                    { id: 'switching-db', title: 'Switching Databases' },
                    { id: 'feature-comparison', title: 'Feature Comparison' }
                  ]
                },
                {
                  id: 'performance',
                  title: 'Performance Considerations',
                  subsections: [
                    { id: 'event-performance', title: 'Event Performance' },
                    { id: 'database-performance', title: 'Database Performance' }
                  ]
                }
              ]} />

              <Card>
                <CardHeader>
                  <Heading size="md">Database Configuration</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      Go-Deployd supports both MongoDB and SQLite databases. Choose based on your deployment needs and scale requirements.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>MongoDB Configuration</Text>
                      <Text mb={2}>Best for: Production environments, horizontal scaling, complex queries</Text>
                      <CodeBlock language="bash">
{`# Set MongoDB connection string
export DATABASE_URL="mongodb://localhost:27017/deployd"

# Or with authentication
export DATABASE_URL="mongodb://username:password@localhost:27017/deployd"

# MongoDB Atlas (cloud)
export DATABASE_URL="mongodb+srv://username:password@cluster.mongodb.net/deployd"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>SQLite Configuration</Text>
                      <Text mb={2}>Best for: Development, testing, small applications, single-server deployments</Text>
                      <CodeBlock language="bash">
{`# Set SQLite database file path
export DATABASE_URL="sqlite:./data/deployd.db"

# Or use in-memory database (for testing)
export DATABASE_URL="sqlite::memory:"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Switching Databases</Text>
                      <Text>
                        Simply change the DATABASE_URL environment variable and restart the server. 
                        Both databases support the same API and event system features.
                      </Text>
                      <CodeBlock language="bash">
{`# Development with SQLite
export DATABASE_URL="sqlite:./data/dev.db"
./deployd

# Production with MongoDB
export DATABASE_URL="mongodb://localhost:27017/deployd_prod"
./deployd`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Feature Comparison</Text>
                      <VStack spacing={2} align="stretch">
                        <Text><strong>MongoDB:</strong></Text>
                        <Text>✅ Horizontal scaling</Text>
                        <Text>✅ Advanced indexing</Text>
                        <Text>✅ Replica sets</Text>
                        <Text>✅ Aggregation pipeline</Text>
                        <Text>❌ Requires separate server</Text>
                        
                        <Text mt={3}><strong>SQLite:</strong></Text>
                        <Text>✅ Zero configuration</Text>
                        <Text>✅ Single file database</Text>
                        <Text>✅ ACID transactions</Text>
                        <Text>✅ Embedded in application</Text>
                        <Text>❌ Single writer limitation</Text>
                      </VStack>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">Performance Considerations</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Box>
                      <Text fontWeight="bold" mb={2}>Event Performance</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>• <strong>Go events:</strong> ~50-100x faster than JavaScript for CPU-intensive tasks</Text>
                        <Text>• <strong>JavaScript events:</strong> Better for simple validations and npm ecosystem</Text>
                        <Text>• Event compilation happens once at startup or file change</Text>
                        <Text>• Use Go events for complex business logic and calculations</Text>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Database Performance</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>• <strong>SQLite:</strong> Excellent for read-heavy workloads, single-server deployments</Text>
                        <Text>• <strong>MongoDB:</strong> Better for write-heavy workloads, multiple servers</Text>
                        <Text>• Both support efficient indexing on collection properties</Text>
                        <Text>• Consider database-specific optimizations in your events</Text>
                      </VStack>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* WebSocket & Real-time Tab */}
          <TabPanel>
            <VStack spacing={6} align="stretch">
              <TableOfContents sections={[
                {
                  id: 'websocket-overview',
                  title: 'WebSocket Real-time Overview',
                  subsections: [
                    { id: 'realtime-features', title: 'Real-time Features' },
                    { id: 'websocket-setup', title: 'WebSocket Setup' },
                    { id: 'connection-scaling', title: 'Multi-Server Scaling' }
                  ]
                },
                {
                  id: 'client-events',
                  title: 'Client-Side Events',
                  subsections: [
                    { id: 'javascript-websocket', title: 'JavaScript WebSocket Client' },
                    { id: 'collection-changes', title: 'Collection Change Events' },
                    { id: 'custom-events', title: 'Custom Events with emit()' }
                  ]
                },
                {
                  id: 'server-events',
                  title: 'Server-Side Events',
                  subsections: [
                    { id: 'aftercommit-events', title: 'AfterCommit Events' },
                    { id: 'emit-from-server', title: 'Emitting from Server Events' },
                    { id: 'event-filtering', title: 'Event Filtering & Routing' }
                  ]
                }
              ]} />

              <Card>
                <CardHeader>
                  <Heading size="md">WebSocket Real-time Overview</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      Go-Deployd provides built-in WebSocket support for real-time applications. Automatically 
                      broadcasts collection changes, supports custom events, and scales across multiple server instances.
                    </Text>
                    
                    <Box>
                      <Text fontWeight="bold" mb={2}>Real-time Features</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>✅ Automatic collection change broadcasting</Text>
                        <Text>✅ Custom event emission from server events</Text>
                        <Text>✅ Multi-server WebSocket scaling with Redis</Text>
                        <Text>✅ Connection pooling and load balancing</Text>
                        <Text>✅ Automatic reconnection and error handling</Text>
                        <Text>✅ Synchronous AfterCommit events with response modification</Text>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>WebSocket Connection</Text>
                      <CodeBlock language="javascript" title="Connect to WebSocket">
{`// Browser WebSocket connection
const ws = new WebSocket('ws://localhost:2403/ws');

ws.onopen = () => {
    console.log('Connected to go-deployd WebSocket');
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Real-time event:', data);
    
    // Handle different event types
    switch(data.type) {
        case 'collection_change':
            handleCollectionChange(data);
            break;
        case 'custom_event':
            handleCustomEvent(data);
            break;
    }
};

function handleCollectionChange(data) {
    console.log(\`Collection: \${data.collection}, Action: \${data.action}\`);
    console.log('Document:', data.document);
    // Update UI accordingly
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Multi-Server Scaling</Text>
                      <Text mb={2}>For production deployments with multiple server instances:</Text>
                      <CodeBlock language="bash" title="Enable Redis for WebSocket scaling">
{`# Set Redis URL for multi-server WebSocket scaling
export REDIS_URL="redis://localhost:6379"

# Start multiple server instances
./deployd --port 2403 &
./deployd --port 2404 &
./deployd --port 2405 &

# WebSocket events will be synchronized across all instances`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">Collection Change Events</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      Collection changes (create, update, delete) are automatically broadcast to all connected 
                      WebSocket clients. No additional configuration required.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Event Structure</Text>
                      <CodeBlock language="json" title="Collection change event format">
{`{
  "type": "collection_change",
  "collection": "todos",
  "action": "created",
  "document": {
    "id": "doc123",
    "title": "New Todo Item",
    "completed": false,
    "createdAt": "2024-06-28T10:00:00Z"
  },
  "timestamp": "2024-06-28T10:00:00Z",
  "userId": "user123"
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Live Collection Monitoring</Text>
                      <CodeBlock language="javascript" title="Monitor specific collections">
{`// Monitor all collection changes
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    if (data.type === 'collection_change') {
        const { collection, action, document } = data;
        
        // Update UI based on collection and action
        switch (collection) {
            case 'todos':
                updateTodoList(action, document);
                break;
            case 'users':
                updateUserList(action, document);
                break;
        }
    }
};

function updateTodoList(action, todo) {
    switch (action) {
        case 'created':
            addTodoToUI(todo);
            break;
        case 'updated':
            updateTodoInUI(todo);
            break;
        case 'deleted':
            removeTodoFromUI(todo.id);
            break;
    }
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Testing Real-time Updates</Text>
                      <Text mb={2}>Create a document to see real-time updates:</Text>
                      <CodeBlock language="bash" title="Trigger real-time event" executable>
{`curl -X POST "${serverUrl}/todos" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: ${masterKey}" \\
  -d '{
    "title": "Real-time test todo",
    "completed": false,
    "priority": 1
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="green.600" mt={2}>
                        💡 Open browser console and connect to WebSocket to see this event broadcast live!
                      </Text>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">Custom Events with emit()</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      Use the <Code>emit()</Code> function in server events to send custom real-time notifications 
                      to connected clients. Perfect for business logic notifications, progress updates, and alerts.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Server-Side Event Emission</Text>
                      <CodeBlock language="javascript" title="JavaScript event with emit()">
{`// aftercommit.js - Emit custom events after successful operations
if (this.priority >= 8) {
    // Emit urgent task alert
    emit('urgent_task_created', {
        taskId: this.id,
        title: this.title,
        priority: this.priority,
        createdBy: me.username,
        timestamp: new Date()
    });
}

if (this.status === 'completed') {
    // Emit completion celebration
    emit('task_completed', {
        taskId: this.id,
        title: this.title,
        completedBy: me.username,
        completionTime: new Date()
    });
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Go Event Emission</Text>
                      <CodeBlock language="go" title="Go event with emit()">
{`// aftercommit.go - Emit custom events from Go
package main

type EventHandler struct{}

func (h *EventHandler) Run(ctx interface{}) error {
    eventCtx := ctx.(*EventContext)
    
    // Check if this is a high-priority task
    if priority, ok := eventCtx.Data["priority"].(float64); ok && priority >= 8 {
        // Emit urgent task alert
        eventCtx.Emit("urgent_task_created", map[string]interface{}{
            "taskId":    eventCtx.Data["id"],
            "title":     eventCtx.Data["title"],
            "priority":  priority,
            "createdBy": eventCtx.Me["username"],
            "timestamp": time.Now(),
        })
    }
    
    // Check if task was completed
    if status, ok := eventCtx.Data["status"].(string); ok && status == "completed" {
        eventCtx.Emit("task_completed", map[string]interface{}{
            "taskId":        eventCtx.Data["id"],
            "title":         eventCtx.Data["title"],
            "completedBy":   eventCtx.Me["username"],
            "completionTime": time.Now(),
        })
    }
    
    return nil
}

var EventHandler = &EventHandler{}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Client-Side Custom Event Handling</Text>
                      <CodeBlock language="javascript" title="Handle custom events on client">
{`// Listen for custom events from server
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    
    if (data.type === 'custom_event') {
        switch (data.event) {
            case 'urgent_task_created':
                showUrgentAlert(data.data);
                break;
            case 'task_completed':
                showCompletionCelebration(data.data);
                break;
            case 'user_online':
                updateUserStatus(data.data.userId, 'online');
                break;
        }
    }
};

function showUrgentAlert(taskData) {
    // Show urgent notification
    const notification = new Notification('Urgent Task Created!', {
        body: \`\${taskData.title} (Priority: \${taskData.priority})\`,
        icon: '/urgent-icon.png'
    });
}

function showCompletionCelebration(taskData) {
    // Show completion animation
    console.log(\`🎉 \${taskData.completedBy} completed: \${taskData.title}\`);
    // Trigger confetti animation, etc.
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Custom Event Structure</Text>
                      <CodeBlock language="json" title="Custom event format">
{`{
  "type": "custom_event",
  "event": "urgent_task_created",
  "data": {
    "taskId": "doc123",
    "title": "Critical System Alert",
    "priority": 9,
    "createdBy": "admin",
    "timestamp": "2024-06-28T10:00:00Z"
  },
  "timestamp": "2024-06-28T10:00:00Z"
}`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">AfterCommit Events & Response Modification</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      AfterCommit events run synchronously and can modify the HTTP response before it's sent to the client. 
                      This allows for real-time updates while also customizing the API response.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Response Modification Example</Text>
                      <CodeBlock language="javascript" title="aftercommit.js - Modify response">
{`// aftercommit.js - Runs AFTER database commit but BEFORE HTTP response
if (this.status === 'completed') {
    // Emit real-time event
    emit('task_completed', {
        taskId: this.id,
        title: this.title,
        completedBy: me.username
    });
    
    // Modify the HTTP response to include additional data
    setResponseData({
        ...this,
        message: 'Congratulations! Task completed successfully!',
        points: calculateCompletionPoints(this.priority),
        nextSuggestion: getNextTaskSuggestion(me.id)
    });
}

// Add real-time notification for high-priority tasks
if (this.priority >= 8) {
    emit('high_priority_task', {
        taskId: this.id,
        title: this.title,
        assignedTo: this.assignedTo
    });
    
    // Add alert to response
    addResponseMessage('High priority task created - notifications sent!');
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Available Response Functions</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>• <Code>setResponseData(object)</Code> - Replace the entire response data</Text>
                        <Text>• <Code>addResponseField(key, value)</Code> - Add a field to the response</Text>
                        <Text>• <Code>addResponseMessage(message)</Code> - Add a message to the response</Text>
                        <Text>• <Code>setResponseStatus(code)</Code> - Change the HTTP status code</Text>
                        <Text>• <Code>emit(event, data)</Code> - Send real-time event to WebSocket clients</Text>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Real-time + Response Example</Text>
                      <Text mb={2}>Test AfterCommit event with response modification:</Text>
                      <CodeBlock language="bash" title="Create high-priority todo" executable>
{`curl -X POST "${serverUrl}/todos" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: ${masterKey}" \\
  -d '{
    "title": "Critical system maintenance",
    "priority": 9,
    "assignedTo": "admin"
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="green.600" mt={2}>
                        💡 Both WebSocket clients and the HTTP response will receive enhanced data!
                      </Text>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* dpd.js Client Tab */}
          <TabPanel>
            <VStack spacing={6} align="stretch">
              <TableOfContents sections={[
                {
                  id: 'dpdjs-overview',
                  title: 'dpd.js Client Library Overview',
                  subsections: [
                    { id: 'installation', title: 'Installation & Setup' },
                    { id: 'basic-usage', title: 'Basic Usage' },
                    { id: 'authentication', title: 'Authentication' }
                  ]
                },
                {
                  id: 'collection-operations',
                  title: 'Collection Operations',
                  subsections: [
                    { id: 'crud-operations', title: 'CRUD Operations' },
                    { id: 'querying', title: 'Advanced Querying' },
                    { id: 'real-time-collections', title: 'Real-time Collection Updates' }
                  ]
                },
                {
                  id: 'advanced-features',
                  title: 'Advanced Features',
                  subsections: [
                    { id: 'file-uploads', title: 'File Uploads' },
                    { id: 'custom-resources', title: 'Custom Resources' },
                    { id: 'error-handling', title: 'Error Handling' }
                  ]
                }
              ]} />

              <Card>
                <CardHeader>
                  <Heading size="md">dpd.js Client Library Overview</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      dpd.js is the official JavaScript client library for go-deployd. It provides a simple, 
                      jQuery-like API for working with collections, real-time events, and authentication.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Installation & Setup</Text>
                      <CodeBlock language="bash" title="Install via npm">
{`# Install dpd.js
npm install dpd

# Or include via CDN
<script src="https://unpkg.com/dpd/dpd.js"></script>`}
                      </CodeBlock>
                      
                      <CodeBlock language="javascript" title="Basic setup">
{`// Initialize dpd client
const dpd = require('dpd');

// Set the server URL (defaults to current domain)
dpd.setBaseURL('http://localhost:2403');

// Access collections
const todos = dpd.todos;
const users = dpd.users;`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Authentication</Text>
                      <CodeBlock language="javascript" title="Login with dpd.js">
{`// Login with username/password
dpd.users.login({
    username: 'user@example.com',
    password: 'password123'
}, function(result, error) {
    if (error) {
        console.error('Login failed:', error);
    } else {
        console.log('Logged in:', result);
        // User is now authenticated for subsequent requests
    }
});

// Login with master key
dpd.users.login({
    masterKey: 'mk_your_master_key_here'
}, function(result, error) {
    if (error) {
        console.error('Master key login failed:', error);
    } else {
        console.log('Logged in as root:', result);
    }
});

// Check current user
dpd.users.me(function(user) {
    if (user) {
        console.log('Current user:', user);
    } else {
        console.log('Not logged in');
    }
});

// Logout
dpd.users.logout();`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Browser Setup</Text>
                      <CodeBlock language="html" title="HTML page with dpd.js">
{`<!DOCTYPE html>
<html>
<head>
    <title>Go-Deployd App</title>
    <script src="https://unpkg.com/dpd/dpd.js"></script>
</head>
<body>
    <script>
        // dpd is available globally
        console.log('Connected to:', dpd.baseURL);
        
        // Use collections
        dpd.todos.get(function(todos) {
            console.log('Todos:', todos);
        });
    </script>
</body>
</html>`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">Collection Operations</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Box>
                      <Text fontWeight="bold" mb={3}>Basic CRUD Operations</Text>
                      <CodeBlock language="javascript" title="CRUD with dpd.js">
{`// CREATE - Add new document
dpd.todos.post({
    title: 'Learn go-deployd',
    completed: false,
    priority: 5
}, function(result, error) {
    if (error) {
        console.error('Create failed:', error);
    } else {
        console.log('Created todo:', result);
    }
});

// READ - Get all documents
dpd.todos.get(function(todos, error) {
    if (error) {
        console.error('Get failed:', error);
    } else {
        console.log('All todos:', todos);
    }
});

// READ - Get single document by ID
dpd.todos.get('doc123', function(todo, error) {
    if (error) {
        console.error('Get failed:', error);
    } else {
        console.log('Todo:', todo);
    }
});

// UPDATE - Update document
dpd.todos.put('doc123', {
    title: 'Updated title',
    completed: true
}, function(result, error) {
    if (error) {
        console.error('Update failed:', error);
    } else {
        console.log('Updated todo:', result);
    }
});

// DELETE - Remove document
dpd.todos.del('doc123', function(result, error) {
    if (error) {
        console.error('Delete failed:', error);
    } else {
        console.log('Deleted todo');
    }
});`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Advanced Querying</Text>
                      <CodeBlock language="javascript" title="Query with conditions">
{`// Simple filtering
dpd.todos.get({
    completed: false,
    priority: {$gte: 5}
}, function(todos) {
    console.log('High priority incomplete todos:', todos);
});

// Complex queries with $or
dpd.todos.get({
    $or: [
        {priority: {$gte: 8}},
        {title: {$regex: 'urgent'}}
    ]
}, function(todos) {
    console.log('Urgent todos:', todos);
});

// Sorting and pagination
dpd.todos.get({
    $sort: {createdAt: -1}, // Sort by created date, newest first
    $limit: 10,             // Limit to 10 results
    $skip: 0                // Skip 0 (for pagination)
}, function(todos) {
    console.log('Recent todos:', todos);
});

// Field projection
dpd.todos.get({
    completed: false,
    $fields: {title: 1, priority: 1} // Only return title and priority
}, function(todos) {
    console.log('Minimal todo data:', todos);
});`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Real-time Collection Updates</Text>
                      <CodeBlock language="javascript" title="Listen for real-time changes">
{`// Listen for real-time updates on todos collection
dpd.todos.on('created', function(todo) {
    console.log('New todo created:', todo);
    addTodoToUI(todo);
});

dpd.todos.on('updated', function(todo) {
    console.log('Todo updated:', todo);
    updateTodoInUI(todo);
});

dpd.todos.on('deleted', function(todo) {
    console.log('Todo deleted:', todo);
    removeTodoFromUI(todo.id);
});

// Listen for all changes
dpd.todos.on('changed', function(todo, action) {
    console.log(\`Todo \${action}:\`, todo);
    switch(action) {
        case 'created':
            addTodoToUI(todo);
            break;
        case 'updated':
            updateTodoInUI(todo);
            break;
        case 'deleted':
            removeTodoFromUI(todo.id);
            break;
    }
});

// Stop listening
dpd.todos.off('created');
dpd.todos.off(); // Remove all listeners`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Promise-based API</Text>
                      <CodeBlock language="javascript" title="Using Promises and async/await">
{`// dpd.js supports both callbacks and promises

// Using Promises
dpd.todos.get({completed: false})
    .then(todos => {
        console.log('Incomplete todos:', todos);
        return dpd.todos.post({
            title: 'New todo from promise',
            completed: false
        });
    })
    .then(newTodo => {
        console.log('Created:', newTodo);
    })
    .catch(error => {
        console.error('Error:', error);
    });

// Using async/await
async function manageTodos() {
    try {
        // Get all incomplete todos
        const incompleteTodos = await dpd.todos.get({completed: false});
        console.log('Found', incompleteTodos.length, 'incomplete todos');
        
        // Create a new todo
        const newTodo = await dpd.todos.post({
            title: 'New todo from async/await',
            completed: false,
            priority: 3
        });
        
        console.log('Created todo:', newTodo);
        
        // Update it
        const updatedTodo = await dpd.todos.put(newTodo.id, {
            completed: true
        });
        
        console.log('Completed todo:', updatedTodo);
        
    } catch (error) {
        console.error('Error managing todos:', error);
    }
}`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">Advanced Features</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Box>
                      <Text fontWeight="bold" mb={3}>Error Handling</Text>
                      <CodeBlock language="javascript" title="Comprehensive error handling">
{`// Handle validation errors
dpd.todos.post({
    title: '', // Invalid - too short
    priority: 'high' // Invalid - should be number
}, function(result, error) {
    if (error) {
        if (error.statusCode === 400) {
            console.log('Validation errors:');
            error.errors.forEach(err => {
                console.log(\`\${err.field}: \${err.message}\`);
            });
        } else {
            console.error('Unexpected error:', error);
        }
    }
});

// Handle authentication errors
dpd.todos.get(function(todos, error) {
    if (error) {
        if (error.statusCode === 401) {
            console.log('Authentication required');
            // Redirect to login
        } else if (error.statusCode === 403) {
            console.log('Access forbidden');
        }
    }
});

// Global error handling
dpd.on('error', function(error) {
    console.error('Global dpd error:', error);
    if (error.statusCode === 401) {
        // Redirect to login page
        window.location.href = '/login';
    }
});`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Custom Events</Text>
                      <CodeBlock language="javascript" title="Listen for custom server events">
{`// Listen for custom events emitted from server events
dpd.on('urgent_task_created', function(data) {
    console.log('Urgent task alert:', data);
    showNotification('Urgent Task!', data.title);
});

dpd.on('task_completed', function(data) {
    console.log('Task completed:', data);
    showCelebration(data.title);
});

// Send custom events to server (if custom endpoint exists)
dpd.notifications.post({
    type: 'user_action',
    action: 'page_view',
    page: window.location.pathname
});`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Complete Example App</Text>
                      <CodeBlock language="javascript" title="Simple todo app with dpd.js">
{`// Simple todo app using dpd.js
class TodoApp {
    constructor() {
        this.todoList = document.getElementById('todo-list');
        this.todoForm = document.getElementById('todo-form');
        this.setupEventListeners();
        this.loadTodos();
    }
    
    setupEventListeners() {
        // Form submission
        this.todoForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.addTodo();
        });
        
        // Real-time updates
        dpd.todos.on('created', (todo) => {
            this.addTodoToDOM(todo);
        });
        
        dpd.todos.on('updated', (todo) => {
            this.updateTodoInDOM(todo);
        });
        
        dpd.todos.on('deleted', (todo) => {
            this.removeTodoFromDOM(todo.id);
        });
    }
    
    async loadTodos() {
        try {
            const todos = await dpd.todos.get({
                $sort: {createdAt: -1}
            });
            
            this.todoList.innerHTML = '';
            todos.forEach(todo => this.addTodoToDOM(todo));
        } catch (error) {
            console.error('Failed to load todos:', error);
        }
    }
    
    async addTodo() {
        const title = document.getElementById('todo-title').value;
        const priority = parseInt(document.getElementById('todo-priority').value);
        
        try {
            await dpd.todos.post({
                title: title,
                completed: false,
                priority: priority
            });
            
            // Clear form
            this.todoForm.reset();
        } catch (error) {
            console.error('Failed to create todo:', error);
            this.showError(error);
        }
    }
    
    addTodoToDOM(todo) {
        const li = document.createElement('li');
        li.dataset.id = todo.id;
        li.innerHTML = \`
            <span class="title">\${todo.title}</span>
            <span class="priority">Priority: \${todo.priority}</span>
            <button onclick="app.toggleComplete('\${todo.id}', \${todo.completed})">
                \${todo.completed ? 'Undo' : 'Complete'}
            </button>
            <button onclick="app.deleteTodo('\${todo.id}')">Delete</button>
        \`;
        
        if (todo.completed) {
            li.classList.add('completed');
        }
        
        this.todoList.appendChild(li);
    }
    
    async toggleComplete(id, currentStatus) {
        try {
            await dpd.todos.put(id, {
                completed: !currentStatus
            });
        } catch (error) {
            console.error('Failed to update todo:', error);
        }
    }
    
    async deleteTodo(id) {
        try {
            await dpd.todos.del(id);
        } catch (error) {
            console.error('Failed to delete todo:', error);
        }
    }
    
    updateTodoInDOM(todo) {
        const li = this.todoList.querySelector(\`li[data-id="\${todo.id}"]\`);
        if (li) {
            li.querySelector('.title').textContent = todo.title;
            li.querySelector('.priority').textContent = \`Priority: \${todo.priority}\`;
            li.classList.toggle('completed', todo.completed);
        }
    }
    
    removeTodoFromDOM(id) {
        const li = this.todoList.querySelector(\`li[data-id="\${id}"]\`);
        if (li) {
            li.remove();
        }
    }
    
    showError(error) {
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error';
        errorDiv.textContent = error.message || 'An error occurred';
        document.body.appendChild(errorDiv);
        
        setTimeout(() => errorDiv.remove(), 5000);
    }
}

// Initialize app when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.app = new TodoApp();
});`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Advanced Queries Tab */}
          <TabPanel>
            <VStack spacing={6} align="stretch">
              <TableOfContents sections={[
                {
                  id: 'query-translation',
                  title: 'MongoDB-to-SQL Query Translation',
                  subsections: [
                    { id: 'supported-operators', title: 'Supported MongoDB Operators' },
                    { id: 'complex-queries', title: 'Complex Nested Queries' },
                    { id: 'query-examples', title: 'Query Examples' }
                  ]
                },
                {
                  id: 'force-mongo',
                  title: '$forceMongo Option',
                  subsections: [
                    { id: 'when-to-use', title: 'When to Use $forceMongo' },
                    { id: 'force-mongo-examples', title: 'Examples with $forceMongo' },
                    { id: 'performance-notes', title: 'Performance Considerations' }
                  ]
                },
                {
                  id: 'post-query-endpoint',
                  title: 'POST /collection/query Endpoint',
                  subsections: [
                    { id: 'endpoint-overview', title: 'Overview' },
                    { id: 'request-format', title: 'Request Format' },
                    { id: 'response-format', title: 'Response Format' }
                  ]
                }
              ]} />

              <Card>
                <CardHeader>
                  <Heading size="md">MongoDB-to-SQL Query Translation</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      Go-Deployd automatically translates MongoDB-style queries to SQL when using SQLite or other SQL databases. 
                      This allows you to use familiar MongoDB query syntax regardless of your backend database.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Supported MongoDB Operators</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>✅ <Code>$eq</Code> - Equal to</Text>
                        <Text>✅ <Code>$ne</Code> - Not equal to</Text>
                        <Text>✅ <Code>$gt</Code> - Greater than</Text>
                        <Text>✅ <Code>$gte</Code> - Greater than or equal</Text>
                        <Text>✅ <Code>$lt</Code> - Less than</Text>
                        <Text>✅ <Code>$lte</Code> - Less than or equal</Text>
                        <Text>✅ <Code>$in</Code> - Value in array</Text>
                        <Text>✅ <Code>$nin</Code> - Value not in array</Text>
                        <Text>✅ <Code>$regex</Code> - Regular expression (converted to LIKE)</Text>
                        <Text>✅ <Code>$exists</Code> - Field exists (IS NOT NULL / IS NULL)</Text>
                        <Text>✅ <Code>$or</Code> - Logical OR</Text>
                        <Text>✅ <Code>$and</Code> - Logical AND</Text>
                        <Text>⚠️ <Code>$nor</Code> - Not yet supported</Text>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Simple Query Translation Examples</Text>
                      <CodeBlock language="bash" title="Basic operators">
{`# Equality
curl "${serverUrl}/todos?status=active"
# Translates to: WHERE JSON_EXTRACT(data, '$.status') = 'active'

# Greater than
curl "${serverUrl}/todos?priority[$gt]=5"
# Translates to: WHERE JSON_EXTRACT(data, '$.priority') > 5

# In array
curl "${serverUrl}/todos?status[$in][]=active&status[$in][]=pending"
# Translates to: WHERE JSON_EXTRACT(data, '$.status') IN ('active', 'pending')

# Regular expression
curl "${serverUrl}/todos?title[$regex]=urgent"
# Translates to: WHERE JSON_EXTRACT(data, '$.title') LIKE '%urgent%'`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Complex Nested Queries</Text>
                      <Text mb={2}>For complex queries, use the POST endpoint with JSON body:</Text>
                      <CodeBlock language="bash" title="Complex OR query" executable>
{`curl -X POST "${serverUrl}/todos/query" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: ${masterKey}" \\
  -d '{
    "query": {
      "$or": [
        {"title": {"$regex": "urgent"}},
        {"priority": {"$gte": 8}}
      ]
    },
    "options": {
      "$limit": 10,
      "$sort": {"createdAt": -1}
    }
  }'`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Advanced Query Examples</Text>
                      <CodeBlock language="json" title="E-commerce product search">
{`{
  "query": {
    "$and": [
      {
        "$or": [
          {"title": {"$regex": "laptop"}},
          {"title": {"$regex": "computer"}}
        ]
      },
      {
        "price": {
          "$gte": 500,
          "$lte": 2000
        }
      },
      {
        "status": {
          "$in": ["available", "limited"]
        }
      }
    ]
  },
  "options": {
    "$sort": {"price": 1},
    "$limit": 20,
    "$fields": {"title": 1, "price": 1, "status": 1}
  }
}`}
                      </CodeBlock>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">$forceMongo Option</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      The <Code>$forceMongo</Code> option bypasses SQL translation and executes queries directly 
                      using MongoDB-style operations. Use this when you need exact MongoDB behavior or when 
                      the SQL translation doesn't support your specific query pattern.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={2}>When to Use $forceMongo</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>✅ Complex aggregation pipelines</Text>
                        <Text>✅ MongoDB-specific operators not yet translated</Text>
                        <Text>✅ When you need exact MongoDB semantics</Text>
                        <Text>✅ Testing queries against MongoDB directly</Text>
                        <Text>⚠️ Only works with stores that support raw MongoDB queries</Text>
                      </VStack>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Using $forceMongo</Text>
                      <CodeBlock language="bash" title="Force MongoDB query execution" executable>
{`curl -X POST "${serverUrl}/todos/query" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: ${masterKey}" \\
  -d '{
    "query": {
      "$or": [
        {"title": {"$regex": "important"}},
        {"tags": {"$elemMatch": {"$eq": "urgent"}}}
      ]
    },
    "options": {
      "$limit": 5,
      "$forceMongo": true
    }
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="blue.600" mt={2}>
                        💡 With $forceMongo: true, the query bypasses SQL translation and runs as pure MongoDB
                      </Text>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Comparison: Translation vs Force Mongo</Text>
                      <CodeBlock language="javascript" title="Same query, different execution">
{`// With SQL translation (default)
{
  "query": {
    "title": {"$regex": "urgent"},
    "priority": {"$gte": 5}
  }
}
// Executes as: WHERE JSON_EXTRACT(data, '$.title') LIKE '%urgent%' 
//              AND JSON_EXTRACT(data, '$.priority') >= 5

// With $forceMongo: true
{
  "query": {
    "title": {"$regex": "urgent"},
    "priority": {"$gte": 5}
  },
  "options": {
    "$forceMongo": true
  }
}
// Executes as native MongoDB query: 
// db.collection.find({title: {$regex: "urgent"}, priority: {$gte: 5}})`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Advanced MongoDB Features</Text>
                      <CodeBlock language="json" title="Complex MongoDB-only operations">
{`{
  "query": {
    "$expr": {
      "$gt": [
        {"$multiply": ["$price", "$quantity"]},
        1000
      ]
    }
  },
  "options": {
    "$forceMongo": true,
    "$limit": 10
  }
}

// Array operations that work better with native MongoDB
{
  "query": {
    "tags": {
      "$elemMatch": {
        "$and": [
          {"$gte": "2024-01-01"},
          {"$lte": "2024-12-31"}
        ]
      }
    }
  },
  "options": {
    "$forceMongo": true
  }
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={2}>Performance Considerations</Text>
                      <VStack spacing={1} align="stretch">
                        <Text>• <Code>$forceMongo</Code> requires store support for raw MongoDB queries</Text>
                        <Text>• SQL translation is often faster for simple queries</Text>
                        <Text>• Use $forceMongo for complex operations that don't translate well</Text>
                        <Text>• Consider indexing strategy when using raw MongoDB queries</Text>
                      </VStack>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>

              <Card>
                <CardHeader>
                  <Heading size="md">POST /collection/query Endpoint</Heading>
                </CardHeader>
                <CardBody>
                  <VStack spacing={4} align="stretch">
                    <Text>
                      The <Code>POST /{"{collection}"}/query</Code> endpoint accepts complex MongoDB-style queries 
                      in the request body, supporting features that can't be expressed in URL query parameters.
                    </Text>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Request Format</Text>
                      <CodeBlock language="json" title="Complete request structure">
{`{
  "query": {
    // MongoDB-style query object
    "$and": [
      {"status": "active"},
      {"priority": {"$gte": 5}}
    ]
  },
  "options": {
    // Query options
    "$sort": {"createdAt": -1},
    "$limit": 20,
    "$skip": 0,
    "$fields": {"title": 1, "status": 1, "priority": 1},
    "$forceMongo": false,
    "$skipEvents": true
  }
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Frontend Integration</Text>
                      <CodeBlock language="javascript" title="Using from JavaScript">
{`// Simple query
const simpleQuery = {
  query: {
    completed: false,
    priority: {$gte: 5}
  },
  options: {
    $limit: 10,
    $sort: {createdAt: -1}
  }
};

const response = await fetch('/todos/query', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': \`Bearer \${token}\`
  },
  body: JSON.stringify(simpleQuery)
});

const todos = await response.json();

// Complex query with OR conditions
const complexQuery = {
  query: {
    $or: [
      {title: {$regex: 'urgent'}},
      {priority: {$gte: 8}},
      {dueDate: {$lt: new Date()}}
    ]
  },
  options: {
    $sort: {priority: -1, dueDate: 1},
    $limit: 50
  }
};

// With dpd.js (if available)
dpd.todos.query(complexQuery, function(results, error) {
  if (error) {
    console.error('Query failed:', error);
  } else {
    console.log('Query results:', results);
  }
});`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Response Format</Text>
                      <CodeBlock language="json" title="Query response">
{`// Success response
[
  {
    "id": "doc123",
    "title": "Urgent task",
    "priority": 9,
    "status": "active",
    "createdAt": "2024-06-28T10:00:00Z",
    "updatedAt": "2024-06-28T10:00:00Z"
  },
  {
    "id": "doc124",
    "title": "High priority item",
    "priority": 8,
    "status": "active",
    "createdAt": "2024-06-28T09:00:00Z",
    "updatedAt": "2024-06-28T09:00:00Z"
  }
]

// Error response
{
  "error": "Query validation failed",
  "details": "Invalid operator: $invalidOp",
  "statusCode": 400
}`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Complete Example: Advanced Search</Text>
                      <CodeBlock language="bash" title="Product search with multiple criteria" executable>
{`curl -X POST "${serverUrl}/todos/query" \\
  -H "Content-Type: application/json" \\
  -H "X-Master-Key: ${masterKey}" \\
  -d '{
    "query": {
      "$and": [
        {
          "$or": [
            {"title": {"$regex": "important"}},
            {"description": {"$regex": "critical"}}
          ]
        },
        {
          "priority": {"$gte": 5}
        },
        {
          "status": {"$in": ["active", "pending"]}
        }
      ]
    },
    "options": {
      "$sort": {"priority": -1, "createdAt": -1},
      "$limit": 15,
      "$fields": {
        "title": 1,
        "priority": 1,
        "status": 1,
        "createdAt": 1
      }
    }
  }'`}
                      </CodeBlock>
                      <Text fontSize="sm" color="green.600" mt={2}>
                        💡 This query finds important/critical todos with priority ≥ 5, sorted by priority and date
                      </Text>
                    </Box>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

        </TabPanels>
      </Tabs>

      {/* Request Editor Modal */}
      <Modal isOpen={isEditorOpen} onClose={onEditorClose} size="4xl">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Edit Request</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack align="stretch" spacing={4}>
              <HStack spacing={4}>
                <FormControl maxW="120px">
                  <FormLabel>Method</FormLabel>
                  <Select value={editMethod} onChange={(e) => setEditMethod(e.target.value)}>
                    <option value="GET">GET</option>
                    <option value="POST">POST</option>
                    <option value="PUT">PUT</option>
                    <option value="DELETE">DELETE</option>
                  </Select>
                </FormControl>
                
                <FormControl flex="1">
                  <FormLabel>URL</FormLabel>
                  <Input
                    value={editUrl}
                    onChange={(e) => setEditUrl(e.target.value)}
                    placeholder="/collections/todos/doc123"
                  />
                </FormControl>
              </HStack>

              <FormControl>
                <FormLabel>Headers (JSON)</FormLabel>
                <Textarea
                  value={editHeaders}
                  onChange={(e) => setEditHeaders(e.target.value)}
                  placeholder='{"Content-Type": "application/json", "X-Master-Key": "..."}'
                  fontFamily="mono"
                  fontSize="sm"
                  rows={6}
                />
              </FormControl>

              {editMethod !== 'GET' && (
                <FormControl>
                  <FormLabel>Body (JSON)</FormLabel>
                  <Textarea
                    value={editBody}
                    onChange={(e) => setEditBody(e.target.value)}
                    placeholder='{"title": "Updated Document", "id": "doc123"}'
                    fontFamily="mono"
                    fontSize="sm"
                    rows={8}
                  />
                </FormControl>
              )}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onEditorClose}>
              Cancel
            </Button>
            <Button 
              colorScheme="green" 
              onClick={executeEditedRequest}
              isLoading={responseLoading}
              loadingText="Executing"
              leftIcon={<FiPlay />}
            >
              Execute Request
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
        </VStack>
      </Box>
    </Box>
  )
}

export default Documentation