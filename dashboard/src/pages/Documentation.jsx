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
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
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
} from 'react-icons/fi'
import { useAuth } from '../contexts/AuthContext'

function Documentation() {
  const [collections, setCollections] = useState([])
  const [selectedCollection, setSelectedCollection] = useState('users')
  const [loading, setLoading] = useState(false)
  const [serverUrl, setServerUrl] = useState('')
  const [masterKey, setMasterKey] = useState('')
  
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

  const CodeBlock = ({ children, language = 'bash', title }) => (
    <Box position="relative" mb={4}>
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
        <Box p={4} fontSize="sm" overflow="auto" maxH="400px" fontFamily="mono" whiteSpace="pre">
          <Code>{children}</Code>
        </Box>
      </Box>
    </Box>
  )

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

  return (
    <VStack align="stretch" spacing={6}>
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

      <Tabs>
        <TabList>
          <Tab><HStack><FiDatabase /><Text>Collections API</Text></HStack></Tab>
          <Tab><HStack><FiKey /><Text>Master Key Auth</Text></HStack></Tab>
          <Tab><HStack><FiUsers /><Text>User Management</Text></HStack></Tab>
          <Tab><HStack><FiShield /><Text>Authentication</Text></HStack></Tab>
          <Tab><HStack><FiServer /><Text>Admin API</Text></HStack></Tab>
        </TabList>

        <TabPanels>
          {/* Collections API Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <Alert status="info">
                <AlertIcon />
                <Box>
                  <AlertTitle>Collection API!</AlertTitle>
                  <AlertDescription>
                    Complete REST API for your collections with MongoDB-style queries, filtering, and more.
                    Select a collection above to see examples.
                  </AlertDescription>
                </Box>
              </Alert>

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
                      <CodeBlock language="bash" title="Get all documents">
{`curl -X GET "${serverUrl}/${selectedCollection}"`}
                      </CodeBlock>
                      <Text fontSize="sm" color="gray.600" mb={2}>Response:</Text>
                      <CodeBlock language="json">
{`[
  {
    "id": "doc123",
    "title": "Example Document",
    "createdAt": "2024-06-22T10:00:00Z",
    "updatedAt": "2024-06-22T10:00:00Z"
  }
]`}
                      </CodeBlock>
                    </Box>

                    {/* GET Single Document */}
                    <Box>
                      <HStack mb={3}>
                        <HttpMethodBadge method="GET" />
                        <Text fontWeight="bold">Get Single Document</Text>
                      </HStack>
                      <CodeBlock language="bash" title="Get document by ID">
{`curl -X GET "${serverUrl}/${selectedCollection}/doc123"`}
                      </CodeBlock>
                    </Box>

                    {/* POST Create Document */}
                    <Box>
                      <HStack mb={3}>
                        <HttpMethodBadge method="POST" />
                        <Text fontWeight="bold">Create Document</Text>
                      </HStack>
                      <CodeBlock language="bash" title="Create new document">
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
              <Alert status="warning">
                <AlertIcon />
                <Box>
                  <AlertTitle>Master Key Security!</AlertTitle>
                  <AlertDescription>
                    The master key provides full administrative access. Keep it secure and never expose it in client-side code.
                  </AlertDescription>
                </Box>
              </Alert>

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
                      <CodeBlock language="bash" title="X-Master-Key header">
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
              <Alert status="info">
                <AlertIcon />
                <Box>
                  <AlertTitle>User Management API</AlertTitle>
                  <AlertDescription>
                    Create and manage users programmatically. When registration is disabled, 
                    only master key holders can create users.
                  </AlertDescription>
                </Box>
              </Alert>

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
              <Alert status="success">
                <AlertIcon />
                <Box>
                  <AlertTitle>Secure by Default!</AlertTitle>
                  <AlertDescription>
                    Go-Deployd uses industry-standard security practices including bcrypt password hashing (cost 12),
                    secure session management, and master key protection.
                  </AlertDescription>
                </Box>
              </Alert>

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
              <Alert status="warning">
                <AlertIcon />
                <Box>
                  <AlertTitle>Admin API Access</AlertTitle>
                  <AlertDescription>
                    All admin API endpoints require master key authentication. These provide full system management capabilities.
                  </AlertDescription>
                </Box>
              </Alert>

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
        </TabPanels>
      </Tabs>
    </VStack>
  )
}

export default Documentation