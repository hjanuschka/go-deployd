import React, { useState, useRef, useEffect } from 'react'
import {
  Box,
  VStack,
  HStack,
  Heading,
  Text,
  Card,
  CardBody,
  CardHeader,
  FormControl,
  FormLabel,
  Select,
  Input,
  Textarea,
  Button,
  Badge,
  useToast,
  Divider,
  Code,
  Alert,
  AlertIcon,
  AlertDescription,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  IconButton,
  useClipboard,
  useColorModeValue,
} from '@chakra-ui/react'
import {
  FiPlay,
  FiCopy,
  FiTrash2,
  FiSave,
  FiDownload,
  FiDatabase,
  FiRefreshCw,
} from 'react-icons/fi'
import { apiService } from '../services/api'

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE']

function ApiTester() {
  const [method, setMethod] = useState('GET')
  const [url, setUrl] = useState('/')
  const [headers, setHeaders] = useState('{}')
  const [body, setBody] = useState('')
  const [response, setResponse] = useState(null)
  const [loading, setLoading] = useState(false)
  const [history, setHistory] = useState([])
  const [collections, setCollections] = useState([])
  const [selectedCollection, setSelectedCollection] = useState('')
  const [collectionSchema, setCollectionSchema] = useState(null)
  
  const toast = useToast()
  const responseRef = useRef()
  const { onCopy } = useClipboard(response ? JSON.stringify(response, null, 2) : '')

  useEffect(() => {
    loadCollections()
  }, [])

  const loadCollections = async () => {
    try {
      const data = await apiService.getCollections()
      setCollections(data || [])
      if (data && data.length > 0) {
        setSelectedCollection(data[0].name)
        setUrl(`/${data[0].name}`)
        loadCollectionSchema(data[0].name)
      }
    } catch (err) {
      console.error('Failed to load collections:', err)
    }
  }

  const loadCollectionSchema = async (collectionName) => {
    try {
      const schema = await apiService.getCollection(collectionName)
      setCollectionSchema(schema)
    } catch (err) {
      console.error('Failed to load collection schema:', err)
      setCollectionSchema(null)
    }
  }

  const handleCollectionChange = (collectionName) => {
    setSelectedCollection(collectionName)
    setUrl(`/${collectionName}`)
    loadCollectionSchema(collectionName)
    generateSampleBody(collectionName)
  }

  const generateSampleBody = (collectionName) => {
    if (!collectionSchema?.properties) return
    
    const sampleData = {}
    Object.entries(collectionSchema.properties).forEach(([key, prop]) => {
      if (key === 'id' || key === 'createdAt' || key === 'updatedAt') return
      
      switch (prop.type) {
        case 'string':
          sampleData[key] = prop.default || `Sample ${key}`
          break
        case 'number':
          sampleData[key] = prop.default || 1
          break
        case 'boolean':
          sampleData[key] = prop.default !== undefined ? prop.default : true
          break
        case 'date':
          sampleData[key] = new Date().toISOString()
          break
        default:
          if (prop.default !== undefined) {
            sampleData[key] = prop.default
          }
      }
    })
    
    if (Object.keys(sampleData).length > 0) {
      setBody(JSON.stringify(sampleData, null, 2))
    }
  }

  const generateSampleBodyString = () => {
    if (!collectionSchema?.properties) {
      return '{\n  "title": "Sample Item",\n  "description": "Sample description"\n}'
    }
    
    const sampleData = {}
    Object.entries(collectionSchema.properties).forEach(([key, prop]) => {
      if (key === 'id' || key === 'createdAt' || key === 'updatedAt') return
      
      switch (prop.type) {
        case 'string':
          sampleData[key] = prop.default || `Sample ${key}`
          break
        case 'number':
          sampleData[key] = prop.default || 1
          break
        case 'boolean':
          sampleData[key] = prop.default !== undefined ? prop.default : true
          break
        case 'date':
          sampleData[key] = new Date().toISOString()
          break
        default:
          if (prop.default !== undefined) {
            sampleData[key] = prop.default
          }
      }
    })
    
    return JSON.stringify(sampleData, null, 2)
  }

  const getPresetRequests = () => {
    const collection = selectedCollection || 'todos'
    return {
      'Get Collections': {
        method: 'GET',
        url: '/_admin/collections',
        headers: '{}',
        body: ''
      },
      [`Get All ${collection}`]: {
        method: 'GET',
        url: `/${collection}`,
        headers: '{}',
        body: ''
      },
      [`Create ${collection}`]: {
        method: 'POST',
        url: `/${collection}`,
        headers: '{"Content-Type": "application/json"}',
        body: generateSampleBodyString()
      },
      [`Update ${collection}`]: {
        method: 'PUT',
        url: `/${collection}/{id}`,
        headers: '{"Content-Type": "application/json"}',
        body: '{\n  "title": "Updated Item"\n}'
      },
      [`Delete ${collection}`]: {
        method: 'DELETE',
        url: `/${collection}/{id}`,
        headers: '{}',
        body: ''
      }
    }
  }

  const executeRequest = async () => {
    if (!url.trim()) {
      toast({
        title: 'URL Required',
        description: 'Please enter a URL to test.',
        status: 'warning',
        duration: 3000,
        isClosable: true,
      })
      return
    }

    setLoading(true)
    const startTime = Date.now()

    try {
      // Parse headers
      let parsedHeaders = {}
      if (headers.trim()) {
        try {
          parsedHeaders = JSON.parse(headers)
        } catch (err) {
          throw new Error('Invalid JSON in headers')
        }
      }

      // Parse body for non-GET requests
      let parsedBody = null
      if (method !== 'GET' && body.trim()) {
        if (parsedHeaders['Content-Type']?.includes('application/json')) {
          try {
            parsedBody = JSON.parse(body)
          } catch (err) {
            throw new Error('Invalid JSON in request body')
          }
        } else {
          parsedBody = body
        }
      }

      // Make the request
      const result = await apiService.testEndpoint(method, url, parsedBody, parsedHeaders)
      const endTime = Date.now()
      
      const responseData = {
        status: result.status,
        statusText: result.statusText,
        headers: result.headers && typeof result.headers.entries === 'function' 
          ? Object.fromEntries(result.headers.entries()) 
          : result.headers || {},
        data: result.data,
        duration: endTime - startTime,
        timestamp: new Date().toISOString()
      }

      setResponse(responseData)

      // Add to history
      const historyItem = {
        id: Date.now(),
        method,
        url,
        headers: parsedHeaders,
        body: parsedBody,
        response: responseData,
        timestamp: new Date().toISOString()
      }
      setHistory(prev => [historyItem, ...prev.slice(0, 9)]) // Keep last 10

      toast({
        title: 'Request Completed',
        description: `${method} ${url} - ${result.status} ${result.statusText}`,
        status: result.status >= 200 && result.status < 300 ? 'success' : 'error',
        duration: 3000,
        isClosable: true,
      })

    } catch (err) {
      const errorResponse = {
        error: true,
        message: err.message,
        duration: Date.now() - startTime,
        timestamp: new Date().toISOString()
      }
      
      if (err.response) {
        errorResponse.status = err.response.status
        errorResponse.statusText = err.response.statusText
        errorResponse.data = err.response.data
      }
      
      setResponse(errorResponse)
      
      toast({
        title: 'Request Failed',
        description: err.message,
        status: 'error',
        duration: 5000,
        isClosable: true,
      })
    } finally {
      setLoading(false)
    }
  }

  const loadPreset = (presetName) => {
    const presetRequests = getPresetRequests()
    const preset = presetRequests[presetName]
    if (preset) {
      setMethod(preset.method)
      setUrl(preset.url)
      setHeaders(preset.headers)
      setBody(preset.body)
    }
  }

  const clearRequest = () => {
    setMethod('GET')
    setUrl('/')
    setHeaders('{}')
    setBody('')
    setResponse(null)
  }

  const formatJSON = (jsonString) => {
    try {
      const parsed = JSON.parse(jsonString)
      return JSON.stringify(parsed, null, 2)
    } catch (err) {
      return jsonString
    }
  }

  const formatHeaders = () => {
    setHeaders(formatJSON(headers))
  }

  const formatBody = () => {
    setBody(formatJSON(body))
  }

  const getStatusColor = (status) => {
    if (status >= 200 && status < 300) return 'green'
    if (status >= 300 && status < 400) return 'blue'
    if (status >= 400 && status < 500) return 'orange'
    if (status >= 500) return 'red'
    return 'gray'
  }

  return (
    <VStack align="stretch" spacing={6}>
      <HStack justify="space-between">
        <Heading size="lg">API Tester</Heading>
        <HStack>
          <Button leftIcon={<FiTrash2 />} variant="outline" size="sm" onClick={clearRequest}>
            Clear
          </Button>
        </HStack>
      </HStack>

      <Card>
        <CardHeader>
          <Heading size="md">Collection Target</Heading>
        </CardHeader>
        <CardBody>
          <HStack spacing={4}>
            <FormControl flex="1">
              <FormLabel>Select Collection</FormLabel>
              <HStack>
                <Select 
                  value={selectedCollection} 
                  onChange={(e) => handleCollectionChange(e.target.value)}
                  placeholder="Choose a collection"
                >
                  {collections.map((collection) => (
                    <option key={collection.name} value={collection.name}>
                      {collection.name}
                    </option>
                  ))}
                </Select>
                <IconButton
                  icon={<FiRefreshCw />}
                  onClick={loadCollections}
                  variant="outline"
                  aria-label="Refresh collections"
                />
              </HStack>
            </FormControl>
            
            {collectionSchema?.properties && (
              <FormControl>
                <FormLabel>Schema Properties</FormLabel>
                <VStack align="start" spacing={1} maxH="100px" overflowY="auto">
                  {Object.entries(collectionSchema.properties).map(([key, prop]) => (
                    <HStack key={key} spacing={2}>
                      <Badge size="sm" colorScheme={prop.required ? "red" : "gray"}>
                        {key}
                      </Badge>
                      <Text fontSize="xs" color="gray.500">
                        {prop.type} {prop.required && "*"}
                      </Text>
                    </HStack>
                  ))}
                </VStack>
              </FormControl>
            )}
            
            <Button
              leftIcon={<FiDatabase />}
              onClick={() => generateSampleBody(selectedCollection)}
              variant="outline"
              size="sm"
            >
              Generate Sample
            </Button>
          </HStack>
        </CardBody>
      </Card>


      <Card>
        <CardHeader>
          <Heading size="md">Quick Presets</Heading>
        </CardHeader>
        <CardBody>
          <HStack wrap="wrap" spacing={2}>
            {Object.keys(getPresetRequests()).map((presetName) => (
              <Button
                key={presetName}
                size="sm"
                variant="outline"
                onClick={() => loadPreset(presetName)}
              >
                {presetName}
              </Button>
            ))}
          </HStack>
        </CardBody>
      </Card>

      <Card>
        <CardHeader>
          <Heading size="md">Request Builder</Heading>
        </CardHeader>
        <CardBody>
          <VStack align="stretch" spacing={4}>
            <HStack>
              <FormControl width="120px">
                <FormLabel>Method</FormLabel>
                <Select value={method} onChange={(e) => setMethod(e.target.value)}>
                  {HTTP_METHODS.map((m) => (
                    <option key={m} value={m}>{m}</option>
                  ))}
                </Select>
              </FormControl>
              
              <FormControl flex="1">
                <FormLabel>URL</FormLabel>
                <Input
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="/todos or /_admin/collections"
                />
              </FormControl>
              
              <Button
                leftIcon={<FiPlay />}
                colorScheme="brand"
                onClick={executeRequest}
                isLoading={loading}
                loadingText="Sending"
                alignSelf="end"
                size="lg"
                px={8}
              >
                Send Request
              </Button>
            </HStack>

            <HStack align="start">
              <FormControl flex="1">
                <HStack justify="space-between">
                  <FormLabel>Headers (JSON)</FormLabel>
                  <Button size="xs" variant="ghost" onClick={formatHeaders}>
                    Format
                  </Button>
                </HStack>
                <Textarea
                  value={headers}
                  onChange={(e) => setHeaders(e.target.value)}
                  placeholder='{"Content-Type": "application/json"}'
                  fontFamily="mono"
                  fontSize="sm"
                  rows={4}
                />
              </FormControl>

              {method !== 'GET' && (
                <FormControl flex="1">
                  <HStack justify="space-between">
                    <FormLabel>Body (JSON)</FormLabel>
                    <Button size="xs" variant="ghost" onClick={formatBody}>
                      Format
                    </Button>
                  </HStack>
                  <Textarea
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    placeholder='{"title": "New Todo", "completed": false}'
                    fontFamily="mono"
                    fontSize="sm"
                    rows={4}
                  />
                </FormControl>
              )}
            </HStack>
          </VStack>
        </CardBody>
      </Card>

      {response && (
        <Card>
          <CardHeader>
            <HStack justify="space-between">
              <Heading size="md">Response</Heading>
              <HStack>
                <IconButton
                  icon={<FiCopy />}
                  size="sm"
                  variant="outline"
                  onClick={onCopy}
                  aria-label="Copy response"
                />
                {response.status && (
                  <Badge colorScheme={getStatusColor(response.status)} variant="solid">
                    {response.status} {response.statusText}
                  </Badge>
                )}
                {response.duration && (
                  <Badge variant="outline">
                    {response.duration}ms
                  </Badge>
                )}
              </HStack>
            </HStack>
          </CardHeader>
          <CardBody>
            <Tabs>
              <TabList>
                <Tab>Response</Tab>
                <Tab>Headers</Tab>
                <Tab>Raw</Tab>
              </TabList>
              <TabPanels>
                <TabPanel>
                  <Box
                    as="pre"
                    bg={useColorModeValue('gray.50', 'gray.800')}
                    p={4}
                    borderRadius="md"
                    overflow="auto"
                    fontSize="sm"
                    fontFamily="mono"
                    maxH="400px"
                    whiteSpace="pre-wrap"
                  >
                    {response.error
                      ? response.message
                      : JSON.stringify(response.data, null, 2)}
                  </Box>
                </TabPanel>
                <TabPanel>
                  <Box
                    as="pre"
                    bg={useColorModeValue('gray.50', 'gray.800')}
                    p={4}
                    borderRadius="md"
                    overflow="auto"
                    fontSize="sm"
                    fontFamily="mono"
                    maxH="400px"
                  >
                    {JSON.stringify(response.headers || {}, null, 2)}
                  </Box>
                </TabPanel>
                <TabPanel>
                  <Box
                    as="pre"
                    bg={useColorModeValue('gray.50', 'gray.800')}
                    p={4}
                    borderRadius="md"
                    overflow="auto"
                    fontSize="sm"
                    fontFamily="mono"
                    maxH="400px"
                  >
                    {JSON.stringify(response, null, 2)}
                  </Box>
                </TabPanel>
              </TabPanels>
            </Tabs>
          </CardBody>
        </Card>
      )}

      {history.length > 0 && (
        <Card>
          <CardHeader>
            <Heading size="md">Request History</Heading>
          </CardHeader>
          <CardBody>
            <VStack align="stretch" spacing={2}>
              {history.map((item) => (
                <HStack
                  key={item.id}
                  p={3}
                  bg={useColorModeValue('gray.50', 'gray.700')}
                  borderRadius="md"
                  justify="space-between"
                  cursor="pointer"
                  _hover={{ bg: useColorModeValue('gray.100', 'gray.600') }}
                  onClick={() => {
                    setMethod(item.method)
                    setUrl(item.url)
                    setHeaders(JSON.stringify(item.headers, null, 2))
                    setBody(item.body ? JSON.stringify(item.body, null, 2) : '')
                    setResponse(item.response)
                  }}
                >
                  <HStack>
                    <Badge colorScheme="blue" variant="solid">
                      {item.method}
                    </Badge>
                    <Code fontSize="sm">{item.url}</Code>
                    <Badge colorScheme={getStatusColor(item.response.status)} variant="outline">
                      {item.response.status}
                    </Badge>
                  </HStack>
                  <Text fontSize="xs" color="gray.500">
                    {new Date(item.timestamp).toLocaleTimeString()}
                  </Text>
                </HStack>
              ))}
            </VStack>
          </CardBody>
        </Card>
      )}
    </VStack>
  )
}

export default ApiTester