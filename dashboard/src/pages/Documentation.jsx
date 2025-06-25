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
  "sessionTTL": 86400,
  "tokenTTL": 2592000,
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
                      <Text fontWeight="bold" mb={3}>Standard Login</Text>
                      <CodeBlock language="bash" title="User login with username/password">
{`curl -X POST "${serverUrl}/users/login" \\
  -H "Content-Type: application/json" \\
  -d '{
    "username": "user@example.com",
    "password": "userpassword"
  }'`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Get Current User (/me)</Text>
                      <CodeBlock language="bash" title="Get current user info">
{`# After login, with session cookie
curl -b cookies.txt "${serverUrl}/users/me"`}
                      </CodeBlock>
                    </Box>

                    <Box>
                      <Text fontWeight="bold" mb={3}>Logout</Text>
                      <CodeBlock language="bash" title="User logout">
{`curl -X POST "${serverUrl}/users/logout" \\
  -b cookies.txt`}
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
                  id: 'security-features',
                  title: 'Security Features'
                },
                {
                  id: 'session-management',
                  title: 'Session Management',
                  subsections: [
                    { id: 'session-properties', title: 'Session Properties' },
                    { id: 'session-validation', title: 'Session Validation' }
                  ]
                }
              ]} />


              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Security Features</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="start" spacing={3}>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>bcrypt password hashing (cost 12)</Text></HStack>
                    <HStack><Badge colorScheme="green">✓</Badge><Text>Secure session management with cookies</Text></HStack>
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
                  <Heading size="md">Session Management</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <Text>
                      Sessions are automatically managed via HTTP-only cookies. User sessions include access control 
                      and filtering based on user permissions.
                    </Text>
                    
                    <Box>
                      <Text fontWeight="bold" mb={2}>Session Properties:</Text>
                      <VStack align="start" spacing={1}>
                        <Text fontSize="sm">• <strong>TTL:</strong> Configurable timeout (default: 24 hours)</Text>
                        <Text fontSize="sm">• <strong>Storage:</strong> Database-backed session store</Text>
                        <Text fontSize="sm">• <strong>Security:</strong> HTTP-only cookies, secure flags in production</Text>
                        <Text fontSize="sm">• <strong>Data:</strong> User ID, role, permissions, login time</Text>
                      </VStack>
                    </Box>

                    <CodeBlock language="bash" title="Session validation">
{`# Sessions are automatically sent with cookies
curl -b cookies.txt "${serverUrl}/users/me"

# Manual session token (if using API tokens)
curl -H "Authorization: Bearer session_token_here" \\
  "${serverUrl}/users/me"`}
                    </CodeBlock>
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
    "sessionTTL": 7200,
    "tokenTTL": 86400,
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
  )
}

export default Documentation