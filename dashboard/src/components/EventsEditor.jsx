import React, { useState, useEffect } from 'react'
import {
  Box,
  VStack,
  HStack,
  Text,
  Button,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Card,
  CardBody,
  Heading,
  Select,
  Badge,
  useToast,
  Alert,
  AlertIcon,
  AlertDescription,
  Link,
  Code,
  IconButton,
  Tooltip,
  useColorModeValue,
} from '@chakra-ui/react'
import CodeMirror from '@uiw/react-codemirror'
import { javascript } from '@codemirror/lang-javascript'
import { go } from '@codemirror/lang-go'
import { oneDark } from '@codemirror/theme-one-dark'
import { EditorView } from '@codemirror/view'
import {
  FiSave,
  FiRefreshCw,
  FiExternalLink,
  FiCode,
  FiFileText,
  FiBook,
} from 'react-icons/fi'
import { apiService } from '../services/api'
import EventTester from './EventTester'
import EventDocumentation from './EventDocumentation'

const EVENT_TYPES = [
  { name: 'get', label: 'On GET', description: 'Runs when fetching documents' },
  { name: 'validate', label: 'On Validate', description: 'Runs before saving (POST/PUT)' },
  { name: 'post', label: 'On POST', description: 'Runs when creating new documents' },
  { name: 'put', label: 'On PUT', description: 'Runs when updating documents' },
  { name: 'delete', label: 'On DELETE', description: 'Runs when deleting documents' },
  { name: 'aftercommit', label: 'After Commit', description: 'Runs after database changes' },
  { name: 'beforerequest', label: 'Before Request', description: 'Runs before any request' },
]

const CODE_TEMPLATES = {
  js: {
    get: `// On GET - Filter or modify retrieved documents
// Available: this (document), me (current user), query, cancel(), error()

// Example: Hide sensitive data from non-admin users
if (!isRoot) {
  hide('internalNotes');
  hide('cost');
}

// Example: Only show user's own documents
if (!isRoot && me) {
  if (this.userId !== me.id) {
    cancel("Not authorized", 403);
  }
}`,
    validate: `// On Validate - Validate data before saving
// Available: this (document), me, error(), hasErrors()

// Example: Validate required fields
if (!this.title || this.title.trim() === '') {
  error('title', 'Title is required');
}

// Example: Custom validation
if (this.priority && (this.priority < 1 || this.priority > 5)) {
  error('priority', 'Priority must be between 1 and 5');
}`,
    post: `// On POST - Modify data when creating documents
// Available: this (document), me, cancel(), error()

// Example: Set defaults
this.createdAt = new Date().toISOString();
this.completed = this.completed || false;

// Example: Add user info
if (me) {
  this.createdBy = me.id;
}

// Example: Require authentication
if (!me) {
  cancel("Authentication required", 401);
}`,
  },
  go: {
    get: `// On GET - Filter or modify retrieved documents (Go version)
func Run(ctx *events.EventContext) error {
    // Hide sensitive fields from non-admin users
    if !ctx.IsRoot {
        ctx.Hide("internalNotes")
        ctx.Hide("cost")
    }
    
    // Only show user's own documents
    if !ctx.IsRoot && ctx.Me != nil {
        if userId, _ := ctx.Data["userId"].(string); userId != ctx.Me["id"] {
            ctx.Cancel("Not authorized", 403)
        }
    }
    
    return nil
}`,
    validate: `// On Validate - Validate data before saving (Go version)
func Run(ctx *events.EventContext) error {
    // Validate required fields
    title, hasTitle := ctx.Data["title"].(string)
    if !hasTitle || strings.TrimSpace(title) == "" {
        ctx.Error("title", "Title is required")
    }
    
    // Custom validation
    if priority, ok := ctx.Data["priority"].(float64); ok {
        if priority < 1 || priority > 5 {
            ctx.Error("priority", "Priority must be between 1 and 5")
        }
    }
    
    return nil
}`,
    post: `// On POST - Modify data when creating documents (Go version)
func Run(ctx *events.EventContext) error {
    // Set defaults
    ctx.Data["createdAt"] = time.Now()
    if _, exists := ctx.Data["completed"]; !exists {
        ctx.Data["completed"] = false
    }
    
    // Add user info
    if ctx.Me != nil {
        ctx.Data["createdBy"] = ctx.Me["id"]
    }
    
    // Require authentication
    if ctx.Me == nil {
        ctx.Cancel("Authentication required", 401)
    }
    
    return nil
}`,
  }
}

function EventsEditor({ collection }) {
  const [events, setEvents] = useState({})
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState({})
  const [scriptType, setScriptType] = useState({})
  const [showDocumentation, setShowDocumentation] = useState(false)
  const toast = useToast()

  useEffect(() => {
    loadEvents()
  }, [collection.name])

  const loadEvents = async () => {
    try {
      setLoading(true)
      // Load event scripts from the API
      const eventData = await apiService.getCollectionEvents(collection.name)
      setEvents(eventData.scripts || {})
      setScriptType(eventData.types || {})
    } catch (err) {
      console.error('Failed to load events:', err)
      // Initialize with empty events
      const emptyEvents = {}
      EVENT_TYPES.forEach(({ name }) => {
        emptyEvents[name] = ''
      })
      setEvents(emptyEvents)
    } finally {
      setLoading(false)
    }
  }

  const handleSaveEvent = async (eventName) => {
    setSaving({ ...saving, [eventName]: true })
    try {
      await apiService.updateCollectionEvent(
        collection.name,
        eventName,
        events[eventName],
        scriptType[eventName] || 'js'
      )
      
      toast({
        title: 'Event saved',
        description: `${eventName} event was saved successfully.`,
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Error saving event',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setSaving({ ...saving, [eventName]: false })
    }
  }

  const handleScriptTypeChange = (eventName, newType) => {
    setScriptType({ ...scriptType, [eventName]: newType })
    
    // If switching to a new type and script is empty, provide a template
    if (!events[eventName] || events[eventName].trim() === '') {
      const template = CODE_TEMPLATES[newType]?.[eventName]
      if (template) {
        setEvents({ ...events, [eventName]: template })
      }
    }
  }

  const insertTemplate = (eventName) => {
    const type = scriptType[eventName] || 'js'
    const template = CODE_TEMPLATES[type]?.[eventName]
    if (template) {
      setEvents({ ...events, [eventName]: template })
    }
  }

  if (loading) {
    return (
      <Box textAlign="center" py={8}>
        <Text color="gray.500">Loading event scripts...</Text>
      </Box>
    )
  }

  return (
    <VStack align="stretch" spacing={4}>
      <HStack justify="space-between">
        <Heading size="md">Event Scripts</Heading>
        <HStack>
          <Button
            leftIcon={<FiBook />}
            variant="outline"
            size="sm"
            onClick={() => setShowDocumentation(true)}
            colorScheme="blue"
          >
            Documentation
          </Button>
          <Button
            leftIcon={<FiRefreshCw />}
            variant="outline"
            size="sm"
            onClick={loadEvents}
          >
            Reload
          </Button>
        </HStack>
      </HStack>

      <Alert status="info" variant="left-accent">
        <AlertIcon />
        <Box>
          <AlertDescription>
            Event scripts run at different stages of the request lifecycle. You can write them in 
            either <Badge mx={1}>JavaScript</Badge> or <Badge mx={1}>Go</Badge> for better performance.
            <Link href="https://docs.deployd.com/docs/collections/adding-logic.html" isExternal ml={2}>
              Learn more <FiExternalLink style={{ display: 'inline' }} />
            </Link>
          </AlertDescription>
        </Box>
      </Alert>

      <Tabs variant="soft-rounded" colorScheme="brand">
        <TabList flexWrap="wrap">
          {EVENT_TYPES.map(({ name, label }) => (
            <Tab key={name} fontSize="sm">{label}</Tab>
          ))}
        </TabList>

        <TabPanels>
          {EVENT_TYPES.map(({ name, label, description }) => (
            <TabPanel key={name}>
              <Card>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <HStack justify="space-between">
                      <Box>
                        <Heading size="sm">{label}</Heading>
                        <Text fontSize="sm" color="gray.500">{description}</Text>
                      </Box>
                      <HStack>
                        <Select
                          size="sm"
                          value={scriptType[name] || 'js'}
                          onChange={(e) => handleScriptTypeChange(name, e.target.value)}
                          width="120px"
                        >
                          <option value="js">JavaScript</option>
                          <option value="go">Go (Hot Reload)</option>
                        </Select>
                        <Tooltip label="Insert template">
                          <IconButton
                            size="sm"
                            icon={<FiFileText />}
                            variant="outline"
                            onClick={() => insertTemplate(name)}
                            aria-label="Insert template"
                          />
                        </Tooltip>
                        <Button
                          size="sm"
                          leftIcon={<FiSave />}
                          colorScheme="brand"
                          onClick={() => handleSaveEvent(name)}
                          isLoading={saving[name]}
                          loadingText="Saving"
                        >
                          {scriptType[name] === 'go' ? 'Hot Reload' : 'Save'}
                        </Button>
                        {scriptType[name] === 'go' && (
                          <Badge colorScheme="green" variant="subtle" fontSize="xs">
                            🔥 Live
                          </Badge>
                        )}
                      </HStack>
                    </HStack>

                    <Box>
                      <HStack mb={2}>
                        <FiCode />
                        <Text fontSize="sm" fontWeight="medium">
                          {name}.{scriptType[name] || 'js'}
                        </Text>
                      </HStack>
                      <Box 
                        border="1px" 
                        borderColor={useColorModeValue('gray.200', 'gray.600')}
                        borderRadius="md"
                        overflow="hidden"
                      >
                        <CodeMirror
                          value={events[name] || ''}
                          onChange={(value) => setEvents({ ...events, [name]: value })}
                          extensions={[
                            scriptType[name] === 'go' ? go() : javascript(),
                            EditorView.theme({
                              '&': {
                                fontSize: '14px',
                              },
                              '.cm-editor': {
                                fontSize: '14px',
                              },
                              '.cm-focused': {
                                outline: 'none',
                              },
                            }),
                          ]}
                          theme={useColorModeValue('light', oneDark)}
                          placeholder={`// ${label} event handler\n// Write your ${scriptType[name] === 'go' ? 'Go' : 'JavaScript'} code here...`}
                          minHeight="400px"
                          basicSetup={{
                            lineNumbers: true,
                            foldGutter: true,
                            dropCursor: false,
                            allowMultipleSelections: false,
                            autocompletion: true,
                            bracketMatching: true,
                            closeBrackets: true,
                            highlightSelectionMatches: false,
                            indentOnInput: true,
                            tabSize: 2,
                          }}
                        />
                      </Box>
                    </Box>

                    <Alert status="info" variant="subtle" fontSize="sm">
                      <AlertIcon />
                      <Box>
                        <Text fontWeight="medium">Available in this event:</Text>
                        <Code fontSize="xs">
                          {scriptType[name] === 'go' 
                            ? 'ctx.Data, ctx.Me, ctx.Query, ctx.Error(), ctx.Cancel()'
                            : 'this, me, query, error(), cancel(), hide()'}
                        </Code>
                      </Box>
                    </Alert>

                    <EventTester 
                      eventName={name} 
                      collection={collection}
                      scriptType={scriptType[name] || 'js'}
                    />
                  </VStack>
                </CardBody>
              </Card>
            </TabPanel>
          ))}
        </TabPanels>
      </Tabs>

      <EventDocumentation 
        isOpen={showDocumentation} 
        onClose={() => setShowDocumentation(false)} 
      />
    </VStack>
  )
}

export default EventsEditor