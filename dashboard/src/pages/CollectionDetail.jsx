import React, { useState, useEffect } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import {
  Box,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Card,
  CardBody,
  CardHeader,
  Text,
  Heading,
  Button,
  IconButton,
  HStack,
  VStack,
  Alert,
  AlertIcon,
  Spinner,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
  Badge,
  useDisclosure,
  useToast,
  useColorModeValue,
} from '@chakra-ui/react'
import {
  FiRefreshCw,
  FiPlus,
  FiEdit,
  FiTrash2,
  FiCode,
  FiChevronUp,
  FiChevronDown,
} from 'react-icons/fi'
import { apiService } from '../services/api'
import DocumentModal from '../components/DocumentModal'
import PropertiesEditor from '../components/PropertiesEditor'
import EventsEditor from '../components/EventsEditor'

function CollectionDetail() {
  const { name } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const [tabIndex, setTabIndex] = useState(() => {
    const tab = searchParams.get('tab')
    if (tab === 'properties') return 1
    if (tab === 'events') return 2
    if (tab === 'api') return 3
    return 0
  })
  const [collection, setCollection] = useState(null)
  const [documents, setDocuments] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [editingDocument, setEditingDocument] = useState(null)
  const [sortColumn, setSortColumn] = useState('id')
  const [sortDirection, setSortDirection] = useState('asc')
  const { isOpen, onOpen, onClose } = useDisclosure()
  const toast = useToast()

  const cardBg = useColorModeValue('white', 'gray.700')
  const borderColor = useColorModeValue('gray.200', 'gray.600')

  // Mock collection data
  const mockCollection = {
    name: 'todos',
    properties: {
      title: { type: 'string', required: true },
      completed: { type: 'boolean', default: false },
      createdAt: { type: 'date', default: 'now' },
      priority: { type: 'number', default: 1 },
      description: { type: 'string' }
    }
  }

  useEffect(() => {
    loadCollectionData()
  }, [name])

  const loadCollectionData = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Get real data from API
      const [collectionData, documentsData] = await Promise.all([
        apiService.getCollection(name).catch(() => mockCollection),
        apiService.getCollectionData(name).catch(() => [])
      ])
      
      setCollection(collectionData)
      setDocuments(documentsData)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleTabChange = (index) => {
    setTabIndex(index)
    const params = new URLSearchParams(searchParams)
    if (index === 1) {
      params.set('tab', 'properties')
    } else if (index === 2) {
      params.set('tab', 'events')
    } else if (index === 3) {
      params.set('tab', 'api')
    } else {
      params.delete('tab')
    }
    setSearchParams(params)
  }

  const handleCreateDocument = () => {
    setEditingDocument(null)
    onOpen()
  }

  const handleEditDocument = (document) => {
    setEditingDocument(document)
    onOpen()
  }

  const handleDeleteDocument = async (id) => {
    if (!confirm('Are you sure you want to delete this document?')) return

    try {
      await apiService.deleteDocument(name, id)
      setDocuments(documents.filter(doc => doc.id !== id))
      
      toast({
        title: 'Document deleted',
        description: 'Document was deleted successfully.',
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Error deleting document',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const handleSaveDocument = async (documentData) => {
    try {
      if (editingDocument) {
        // Update existing document
        const updated = await apiService.updateDocument(name, editingDocument.id, documentData)
        setDocuments(documents.map(doc => 
          doc.id === editingDocument.id ? updated : doc
        ))
        
        toast({
          title: 'Document updated',
          description: 'Document was updated successfully.',
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
      } else {
        // Create new document
        const newDoc = await apiService.createDocument(name, documentData)
        setDocuments([...documents, newDoc])
        
        toast({
          title: 'Document created',
          description: 'Document was created successfully.',
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
      }
      onClose()
    } catch (err) {
      toast({
        title: `Error ${editingDocument ? 'updating' : 'creating'} document`,
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const formatValue = (value, type) => {
    if (value === null || value === undefined) return '-'
    
    switch (type) {
      case 'date':
        const date = new Date(value)
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
      case 'boolean':
        return value ? 'Yes' : 'No'
      case 'object':
      case 'array':
        return JSON.stringify(value)
      default:
        return String(value)
    }
  }

  const getTableColumns = () => {
    if (!collection?.properties) return ['id', 'createdAt', 'updatedAt']
    
    const columns = ['id']
    const properties = Object.keys(collection.properties)
    
    // Add user-defined properties first (excluding timestamps)
    properties.forEach(prop => {
      if (prop !== 'createdAt' && prop !== 'updatedAt') {
        columns.push(prop)
      }
    })
    
    // Always add timestamps at the end
    if (properties.includes('createdAt')) {
      columns.push('createdAt')
    }
    if (properties.includes('updatedAt')) {
      columns.push('updatedAt')
    }
    
    return columns
  }

  const handleSort = (column) => {
    if (sortColumn === column) {
      // Toggle direction if same column
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      // New column, default to ascending
      setSortColumn(column)
      setSortDirection('asc')
    }
  }

  const getSortedDocuments = () => {
    if (!documents.length) return documents

    return [...documents].sort((a, b) => {
      let aValue = a[sortColumn]
      let bValue = b[sortColumn]
      
      // Handle different data types
      if (aValue === null || aValue === undefined) aValue = ''
      if (bValue === null || bValue === undefined) bValue = ''
      
      const columnType = collection?.properties?.[sortColumn]?.type || 'string'
      
      // Type-specific sorting
      switch (columnType) {
        case 'number':
          aValue = Number(aValue) || 0
          bValue = Number(bValue) || 0
          break
        case 'date':
          aValue = new Date(aValue).getTime() || 0
          bValue = new Date(bValue).getTime() || 0
          break
        case 'boolean':
          aValue = Boolean(aValue) ? 1 : 0
          bValue = Boolean(bValue) ? 1 : 0
          break
        default:
          aValue = String(aValue).toLowerCase()
          bValue = String(bValue).toLowerCase()
      }
      
      let result = 0
      if (aValue < bValue) result = -1
      else if (aValue > bValue) result = 1
      
      return sortDirection === 'desc' ? -result : result
    })
  }

  const getSortIcon = (column) => {
    if (sortColumn !== column) {
      return null
    }
    return sortDirection === 'asc' ? <FiChevronUp /> : <FiChevronDown />
  }

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minH="300px">
        <VStack spacing={4}>
          <Spinner size="xl" color="brand.500" />
          <Text>Loading collection...</Text>
        </VStack>
      </Box>
    )
  }

  if (error) {
    return (
      <Alert status="error">
        <AlertIcon />
        <Box flex="1">
          <Text>{error}</Text>
        </Box>
        <IconButton
          icon={<FiRefreshCw />}
          onClick={loadCollectionData}
          size="sm"
          variant="ghost"
        />
      </Alert>
    )
  }

  if (!collection) {
    return (
      <Alert status="error">
        <AlertIcon />
        <Text>Collection not found</Text>
      </Alert>
    )
  }

  return (
    <Box>
      <HStack justify="space-between" mb={6}>
        <VStack align="start" spacing={1}>
          <Heading size="lg">{collection.name}</Heading>
          <Text color="gray.500">{documents.length} documents</Text>
        </VStack>
        <IconButton
          icon={<FiRefreshCw />}
          onClick={loadCollectionData}
          variant="outline"
          aria-label="Refresh data"
        />
      </HStack>

      <Card bg={cardBg} shadow="md">
        <Tabs index={tabIndex} onChange={handleTabChange}>
          <TabList>
            <Tab>Data</Tab>
            <Tab>Properties</Tab>
            <Tab>Events</Tab>
            <Tab>API</Tab>
          </TabList>

          <TabPanels>
            {/* Data Tab */}
            <TabPanel>
              <VStack align="stretch" spacing={4}>
                <HStack justify="space-between">
                  <Heading size="md">Documents ({documents.length})</Heading>
                  <Button
                    leftIcon={<FiPlus />}
                    colorScheme="brand"
                    onClick={handleCreateDocument}
                  >
                    Add Document
                  </Button>
                </HStack>

                <TableContainer>
                  <Table variant="simple" size="sm">
                    <Thead>
                      <Tr>
                        {getTableColumns().map((column) => (
                          <Th 
                            key={column}
                            cursor="pointer"
                            userSelect="none"
                            _hover={{ bg: useColorModeValue('gray.50', 'gray.600') }}
                            onClick={() => handleSort(column)}
                            position="relative"
                          >
                            <HStack spacing={2} justify="space-between">
                              <Text>{column}</Text>
                              <Box 
                                opacity={sortColumn === column ? 1 : 0.3}
                                transition="opacity 0.2s"
                              >
                                {getSortIcon(column) || <FiChevronUp />}
                              </Box>
                            </HStack>
                          </Th>
                        ))}
                        <Th>Actions</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {getSortedDocuments().map((document) => (
                        <Tr key={document.id}>
                          {getTableColumns().map((column) => (
                            <Td key={column} maxW="200px" isTruncated>
                              {column === 'id' ? (
                                <Text fontFamily="mono" fontSize="sm">
                                  {document[column]}
                                </Text>
                              ) : (
                                formatValue(
                                  document[column], 
                                  collection.properties?.[column]?.type
                                )
                              )}
                            </Td>
                          ))}
                          <Td>
                            <HStack spacing={1}>
                              <IconButton
                                size="sm"
                                icon={<FiEdit />}
                                variant="outline"
                                onClick={() => handleEditDocument(document)}
                                aria-label="Edit document"
                              />
                              <IconButton
                                size="sm"
                                icon={<FiTrash2 />}
                                colorScheme="red"
                                variant="outline"
                                onClick={() => handleDeleteDocument(document.id)}
                                aria-label="Delete document"
                              />
                            </HStack>
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </TableContainer>

                {documents.length === 0 && (
                  <Box textAlign="center" py={8}>
                    <Text color="gray.500" mb={4}>No documents yet</Text>
                    <Button
                      leftIcon={<FiPlus />}
                      colorScheme="brand"
                      onClick={handleCreateDocument}
                    >
                      Create your first document
                    </Button>
                  </Box>
                )}
              </VStack>
            </TabPanel>

            {/* Properties Tab */}
            <TabPanel>
              <PropertiesEditor
                collection={collection}
                onUpdate={async (updatedCollection) => {
                  // TODO: Implement collection update API
                  setCollection(updatedCollection)
                }}
              />
            </TabPanel>

            {/* Events Tab */}
            <TabPanel>
              <EventsEditor collection={collection} />
            </TabPanel>

            {/* API Tab */}
            <TabPanel>
              <VStack align="stretch" spacing={4}>
                <Heading size="md">API Endpoints</Heading>
                
                <Card bg={useColorModeValue('gray.50', 'gray.800')} p={4}>
                  <Text fontFamily="mono" fontSize="sm" whiteSpace="pre-line">
{`GET    /${name}           - List all documents
POST   /${name}           - Create new document  
GET    /${name}/{id}      - Get specific document
PUT    /${name}/{id}      - Update document
DELETE /${name}/{id}      - Delete document
GET    /${name}/count     - Count documents`}
                  </Text>
                </Card>

                <Heading size="sm" mt={4}>Example Usage</Heading>
                <Card bg={useColorModeValue('gray.50', 'gray.800')} p={4}>
                  <Text fontFamily="mono" fontSize="sm" whiteSpace="pre-line">
{`# Get all ${name}
curl http://localhost:2403/${name}

# Create a new ${name.slice(0, -1)}
curl -X POST http://localhost:2403/${name} \\
  -H "Content-Type: application/json" \\
  -d '{"title": "Example"}'

# Update a ${name.slice(0, -1)}
curl -X PUT http://localhost:2403/${name}/{id} \\
  -H "Content-Type: application/json" \\
  -d '{"completed": true}'`}
                  </Text>
                </Card>
              </VStack>
            </TabPanel>
          </TabPanels>
        </Tabs>
      </Card>

      {/* Document Modal */}
      <DocumentModal
        isOpen={isOpen}
        onClose={onClose}
        document={editingDocument}
        collection={collection}
        onSave={handleSaveDocument}
      />
    </Box>
  )
}

export default CollectionDetail