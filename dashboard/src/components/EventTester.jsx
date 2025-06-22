import React, { useState } from 'react'
import {
  Box,
  VStack,
  HStack,
  Text,
  Button,
  Card,
  CardBody,
  CardHeader,
  Textarea,
  useToast,
  Alert,
  AlertIcon,
  AlertDescription,
  Badge,
  Collapse,
  useDisclosure,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Code,
  Select,
  FormControl,
  FormLabel,
  Divider,
} from '@chakra-ui/react'
import {
  FiPlay,
  FiChevronDown,
  FiChevronUp,
  FiDatabase,
  FiUser,
  FiSettings,
} from 'react-icons/fi'
import { apiService } from '../services/api'

// Generate mock data based on collection schema
const generateMockData = (collection, eventName, preset = 'valid') => {
  if (!collection?.properties) return {}
  
  const data = {}
  const collectionName = collection.name || 'item'
  
  // Add ID for events that need it
  if (['get', 'put', 'delete'].includes(eventName)) {
    data.id = '507f1f77bcf86cd799439' + Math.floor(Math.random() * 1000).toString().padStart(3, '0')
  }
  
  Object.entries(collection.properties).forEach(([fieldName, fieldConfig]) => {
    const { type, required, default: defaultValue } = fieldConfig
    
    // Generate data based on preset type
    switch (preset) {
      case 'valid':
        data[fieldName] = generateValidValue(fieldName, type, defaultValue, collectionName)
        break
      case 'invalid':
        if (required) {
          data[fieldName] = generateInvalidValue(type)
        }
        break
      case 'minimal':
        if (required) {
          data[fieldName] = generateValidValue(fieldName, type, defaultValue, collectionName)
        }
        break
      case 'complete':
        data[fieldName] = generateValidValue(fieldName, type, defaultValue, collectionName)
        break
      case 'update':
        // For PUT events, only include some fields
        if (Math.random() > 0.5) {
          data[fieldName] = generateValidValue(fieldName, type, defaultValue, collectionName)
        }
        break
    }
  })
  
  return data
}

const generateValidValue = (fieldName, type, defaultValue, collectionName) => {
  // Use default value if available
  if (defaultValue !== undefined && defaultValue !== null) {
    if (defaultValue === 'now' && type === 'date') {
      return new Date().toISOString()
    }
    return defaultValue
  }
  
  // Generate based on field name and type
  switch (type) {
    case 'string':
      return generateStringValue(fieldName, collectionName)
    case 'number':
      return generateNumberValue(fieldName)
    case 'boolean':
      return generateBooleanValue(fieldName)
    case 'date':
      return new Date().toISOString()
    case 'array':
      return generateArrayValue(fieldName)
    case 'object':
      return generateObjectValue(fieldName)
    default:
      return generateStringValue(fieldName, collectionName)
  }
}

const generateInvalidValue = (type) => {
  switch (type) {
    case 'string':
      return '' // Empty string
    case 'number':
      return 'not-a-number'
    case 'boolean':
      return 'maybe'
    case 'date':
      return 'invalid-date'
    case 'array':
      return 'not-an-array'
    case 'object':
      return 'not-an-object'
    default:
      return null
  }
}

const generateStringValue = (fieldName, collectionName) => {
  const lowerField = fieldName.toLowerCase()
  const lowerCollection = collectionName.toLowerCase()
  
  if (lowerField.includes('email')) return 'user@example.com'
  if (lowerField.includes('name') || lowerField.includes('title')) {
    return `Sample ${collectionName.slice(0, -1) || 'Item'} ${Math.floor(Math.random() * 100)}`
  }
  if (lowerField.includes('description')) return `This is a sample ${lowerCollection.slice(0, -1)} description`
  if (lowerField.includes('url')) return 'https://example.com'
  if (lowerField.includes('phone')) return '+1-555-123-4567'
  if (lowerField.includes('address')) return '123 Sample St, Example City, EX 12345'
  if (lowerField.includes('status')) return 'active'
  if (lowerField.includes('type') || lowerField.includes('category')) return 'general'
  if (lowerField.includes('code')) return 'SAMPLE123'
  if (lowerField.includes('id') && !lowerField.startsWith('id')) return 'usr_' + Math.random().toString(36).substr(2, 9)
  
  return `Sample ${fieldName}`
}

const generateNumberValue = (fieldName) => {
  const lowerField = fieldName.toLowerCase()
  
  if (lowerField.includes('priority')) return Math.floor(Math.random() * 5) + 1
  if (lowerField.includes('price') || lowerField.includes('cost')) return Math.round(Math.random() * 100 * 100) / 100
  if (lowerField.includes('count') || lowerField.includes('quantity')) return Math.floor(Math.random() * 10) + 1
  if (lowerField.includes('age')) return Math.floor(Math.random() * 50) + 18
  if (lowerField.includes('score') || lowerField.includes('rating')) return Math.floor(Math.random() * 5) + 1
  if (lowerField.includes('percent')) return Math.floor(Math.random() * 100)
  
  return Math.floor(Math.random() * 100)
}

const generateBooleanValue = (fieldName) => {
  const lowerField = fieldName.toLowerCase()
  
  if (lowerField.includes('completed') || lowerField.includes('done')) return Math.random() > 0.7
  if (lowerField.includes('active') || lowerField.includes('enabled')) return Math.random() > 0.3
  if (lowerField.includes('verified') || lowerField.includes('confirmed')) return Math.random() > 0.5
  if (lowerField.includes('public') || lowerField.includes('visible')) return Math.random() > 0.4
  
  return Math.random() > 0.5
}

const generateArrayValue = (fieldName) => {
  const lowerField = fieldName.toLowerCase()
  
  if (lowerField.includes('tag')) return ['sample', 'test', 'demo']
  if (lowerField.includes('category') || lowerField.includes('categories')) return ['general', 'important']
  if (lowerField.includes('skill')) return ['javascript', 'react', 'nodejs']
  if (lowerField.includes('permission')) return ['read', 'write']
  
  return ['item1', 'item2', 'item3']
}

const generateObjectValue = (fieldName) => {
  const lowerField = fieldName.toLowerCase()
  
  if (lowerField.includes('address')) {
    return { street: '123 Sample St', city: 'Example City', zip: '12345' }
  }
  if (lowerField.includes('metadata') || lowerField.includes('meta')) {
    return { category: 'sample', source: 'generated' }
  }
  if (lowerField.includes('settings') || lowerField.includes('config')) {
    return { theme: 'light', notifications: true }
  }
  
  return { sample: 'data', generated: true }
}

const MOCK_USER_CONTEXTS = {
  authenticated: {
    id: 'user123',
    username: 'testuser',
    email: 'test@example.com',
    isAdmin: false
  },
  admin: {
    id: 'admin456',
    username: 'admin',
    email: 'admin@example.com',
    isAdmin: true
  },
  anonymous: null
}

function EventTester({ eventName, collection, scriptType }) {
  const [mockData, setMockData] = useState('')
  const [mockUser, setMockUser] = useState('authenticated')
  const [testResult, setTestResult] = useState(null)
  const [testing, setTesting] = useState(false)
  const { isOpen, onToggle } = useDisclosure()
  const toast = useToast()

  const loadPreset = (presetType) => {
    const generatedData = generateMockData(collection, eventName, presetType)
    if (generatedData && Object.keys(generatedData).length > 0) {
      setMockData(JSON.stringify(generatedData, null, 2))
    }
  }

  const runTest = async () => {
    if (!mockData.trim()) {
      toast({
        title: 'Mock Data Required',
        description: 'Please provide mock data to test the event.',
        status: 'warning',
        duration: 3000,
        isClosable: true,
      })
      return
    }

    setTesting(true)
    try {
      let testData
      try {
        testData = JSON.parse(mockData)
      } catch (err) {
        throw new Error('Invalid JSON in mock data')
      }

      // Test using real API (hot-reload for Go, regular for JS)
      const testContext = {
        data: testData,
        user: MOCK_USER_CONTEXTS[mockUser],
        query: {},
        scriptType
      }

      const response = await apiService.testCollectionEvent(
        collection.name, 
        eventName, 
        testContext
      )
      
      setTestResult({
        success: response.success,
        data: response.data,
        errors: response.errors,
        logs: response.logs,
        duration: response.duration
      })

      toast({
        title: response.success ? 'Test Passed' : 'Test Failed',
        description: response.success 
          ? 'Event executed successfully with mock data'
          : 'Event execution failed',
        status: response.success ? 'success' : 'error',
        duration: 3000,
        isClosable: true,
      })

    } catch (err) {
      setTestResult({
        success: false,
        error: err.message,
        duration: 0
      })

      toast({
        title: 'Test Error',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setTesting(false)
    }
  }

  // Mock event execution simulation
  const simulateEventExecution = async (context) => {
    // Simulate API delay
    await new Promise(resolve => setTimeout(resolve, 500))

    const startTime = Date.now()
    const logs = []
    let success = true
    let errors = {}
    let data = { ...context.data }

    try {
      // Simulate event logic based on event type
      switch (context.eventType) {
        case 'validate':
          if (!data.title || data.title.trim() === '') {
            errors.title = 'Title is required'
            success = false
          }
          if (data.priority && (data.priority < 1 || data.priority > 5)) {
            errors.priority = 'Priority must be between 1 and 5'
            success = false
          }
          logs.push('Validation checks completed')
          break

        case 'post':
          if (!data.title) {
            errors.title = 'Title is required'
            success = false
          } else {
            data.createdAt = new Date().toISOString()
            data.id = '507f1f77bcf86cd799439' + Math.floor(Math.random() * 1000).toString().padStart(3, '0')
            if (context.user) {
              data.createdBy = context.user.id
            }
            logs.push('Document created with generated ID and timestamp')
          }
          break

        case 'put':
          data.updatedAt = new Date().toISOString()
          if (context.user) {
            data.updatedBy = context.user.id
          }
          if (data.completed === true && !data.completedAt) {
            data.completedAt = new Date().toISOString()
            logs.push('Set completion timestamp')
          }
          logs.push('Document updated')
          break

        case 'get':
          if (!context.user) {
            success = false
            errors.auth = 'Authentication required'
          } else if (data.userId && data.userId !== context.user.id && !context.user.isAdmin) {
            success = false
            errors.auth = 'Access denied'
          } else {
            logs.push('Access granted')
          }
          break

        case 'delete':
          if (!context.user) {
            success = false
            errors.auth = 'Authentication required'
          } else {
            logs.push('Document marked for deletion')
          }
          break

        default:
          logs.push(`${context.eventType} event executed`)
      }

    } catch (err) {
      success = false
      errors.runtime = err.message
    }

    return {
      success,
      data,
      errors: Object.keys(errors).length > 0 ? errors : null,
      logs,
      duration: Date.now() - startTime
    }
  }

  return (
    <Card size="sm" variant="outline">
      <CardHeader pb={2}>
        <HStack justify="space-between" cursor="pointer" onClick={onToggle}>
          <HStack>
            <FiPlay size={14} />
            <Text fontSize="sm" fontWeight="medium">Test Event</Text>
          </HStack>
          {isOpen ? <FiChevronUp size={14} /> : <FiChevronDown size={14} />}
        </HStack>
      </CardHeader>
      
      <Collapse in={isOpen}>
        <CardBody pt={0}>
          <VStack align="stretch" spacing={4}>
            <HStack>
              <FormControl size="sm">
                <FormLabel fontSize="xs">Mock User</FormLabel>
                <Select 
                  size="sm" 
                  value={mockUser} 
                  onChange={(e) => setMockUser(e.target.value)}
                >
                  <option value="authenticated">Authenticated User</option>
                  <option value="admin">Admin User</option>
                  <option value="anonymous">Anonymous</option>
                </Select>
              </FormControl>
              
              <FormControl size="sm">
                <FormLabel fontSize="xs">Data Preset</FormLabel>
                <Select 
                  size="sm" 
                  placeholder="Load preset..."
                  onChange={(e) => e.target.value && loadPreset(e.target.value)}
                >
                  <option value="valid">Valid Data</option>
                  <option value="minimal">Minimal (Required Only)</option>
                  <option value="complete">Complete (All Fields)</option>
                  {eventName === 'put' && <option value="update">Partial Update</option>}
                  <option value="invalid">Invalid Data (Test Validation)</option>
                </Select>
              </FormControl>
            </HStack>

            <Box>
              <Text fontSize="xs" fontWeight="medium" mb={2}>Mock Data (JSON)</Text>
              <Textarea
                value={mockData}
                onChange={(e) => setMockData(e.target.value)}
                placeholder={JSON.stringify(generateMockData(collection, eventName, 'minimal'), null, 2)}
                fontFamily="mono"
                fontSize="xs"
                rows={6}
                size="sm"
              />
            </Box>

            <HStack>
              <Button
                leftIcon={<FiPlay />}
                size="sm"
                colorScheme="brand"
                onClick={runTest}
                isLoading={testing}
                loadingText="Testing"
                flex="1"
              >
                Run Test
              </Button>
            </HStack>

            {testResult && (
              <Card size="sm" variant={testResult.success ? 'subtle' : 'outline'}>
                <CardBody>
                  <VStack align="stretch" spacing={3}>
                    <HStack justify="space-between">
                      <HStack>
                        <Badge colorScheme={testResult.success ? 'green' : 'red'}>
                          {testResult.success ? 'PASS' : 'FAIL'}
                        </Badge>
                        {testResult.duration !== undefined && (
                          <Badge variant="outline">{testResult.duration}ms</Badge>
                        )}
                      </HStack>
                    </HStack>

                    {testResult.error && (
                      <Alert status="error" size="sm">
                        <AlertIcon />
                        <AlertDescription fontSize="xs">
                          {testResult.error}
                        </AlertDescription>
                      </Alert>
                    )}

                    {testResult.errors && (
                      <Box>
                        <Text fontSize="xs" fontWeight="medium" mb={1}>Validation Errors:</Text>
                        <Box bg="red.50" p={2} borderRadius="md" border="1px" borderColor="red.200">
                          {Object.entries(testResult.errors).map(([field, message]) => (
                            <Text key={field} fontSize="xs" color="red.700">
                              <Code fontSize="xs">{field}</Code>: {message}
                            </Text>
                          ))}
                        </Box>
                      </Box>
                    )}

                    {testResult.data && (
                      <Box>
                        <Text fontSize="xs" fontWeight="medium" mb={1}>Modified Data:</Text>
                        <Box 
                          as="pre" 
                          bg="gray.50" 
                          p={2} 
                          borderRadius="md" 
                          fontSize="xs" 
                          fontFamily="mono"
                          overflow="auto"
                          maxH="150px"
                        >
                          {JSON.stringify(testResult.data, null, 2)}
                        </Box>
                      </Box>
                    )}

                    {testResult.logs && testResult.logs.length > 0 && (
                      <Box>
                        <Text fontSize="xs" fontWeight="medium" mb={1}>Execution Log:</Text>
                        <VStack align="start" spacing={1}>
                          {testResult.logs.map((log, index) => (
                            <Text key={index} fontSize="xs" color="gray.600">
                              • {log}
                            </Text>
                          ))}
                        </VStack>
                      </Box>
                    )}
                  </VStack>
                </CardBody>
              </Card>
            )}
          </VStack>
        </CardBody>
      </Collapse>
    </Card>
  )
}

export default EventTester