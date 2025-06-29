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
  Flex,
  InputGroup,
  InputLeftElement,
  Input,
  Collapse,
} from '@chakra-ui/react'
import {
  FiRefreshCw,
  FiPlus,
  FiEdit,
  FiTrash2,
  FiCode,
  FiChevronUp,
  FiChevronDown,
  FiChevronLeft,
  FiChevronRight,
  FiSearch,
  FiFilter,
  FiX,
} from 'react-icons/fi'
import { apiService } from '../services/api'
import DocumentModal from '../components/DocumentModal'
import PropertiesEditor from '../components/PropertiesEditor'
import EventsEditor from '../components/EventsEditor'
import VisualQueryBuilder from '../components/VisualQueryBuilder'
import { AnimatedBackground } from '../components/AnimatedBackground'

function CollectionDetail() {
  const { name } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const [tabIndex, setTabIndex] = useState(() => {
    const tab = searchParams.get('tab')
    if (tab === 'query') return 1
    if (tab === 'properties') return 2
    if (tab === 'events') return 3
    if (tab === 'api') return 4
    return 0
  })
  const [collection, setCollection] = useState(null)
  const [documents, setDocuments] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [editingDocument, setEditingDocument] = useState(null)
  const [sortColumn, setSortColumn] = useState('id')
  const [sortDirection, setSortDirection] = useState('asc')
  const [currentPage, setCurrentPage] = useState(1)
  const [documentsPerPage] = useState(50)
  const [currentQuery, setCurrentQuery] = useState({})
  const [queryResults, setQueryResults] = useState([])
  const [queryLoading, setQueryLoading] = useState(false)
  const [searchText, setSearchText] = useState('')
  const [columnFilters, setColumnFilters] = useState({})
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false)
  const { isOpen, onOpen, onClose } = useDisclosure()
  const toast = useToast()

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
      setDocuments(Array.isArray(documentsData) ? documentsData : [])
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
      params.set('tab', 'query')
    } else if (index === 2) {
      params.set('tab', 'properties')
    } else if (index === 3) {
      params.set('tab', 'events')
    } else if (index === 4) {
      params.set('tab', 'api')
    } else {
      params.delete('tab')
    }
    setSearchParams(params)
  }

  // Handler for query execution from VisualQueryBuilder
  const handleQueryExecute = async ({ query, options }) => {
    try {
      setQueryLoading(true)
      setError(null)
      
      // Execute MongoDB-style query via API
      const results = await apiService.queryCollection(name, query, options)
      setQueryResults(results)
      
      toast({
        title: 'Query executed successfully',
        description: `Found ${results.length} matching documents`,
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      setError(err.message)
      toast({
        title: 'Query execution failed',
        description: err.message,
        status: 'error',
        duration: 5000,
        isClosable: true,
      })
    } finally {
      setQueryLoading(false)
    }
  }

  // Filter documents based on search and column filters
  const getFilteredDocuments = () => {
    // Safety check to ensure documents is always an array
    if (!Array.isArray(documents)) {
      return []
    }
    let filtered = [...documents]

    // Apply search text filter
    if (searchText) {
      const searchLower = searchText.toLowerCase()
      filtered = filtered.filter(doc => {
        return Object.values(doc).some(value => {
          if (value == null) return false
          return String(value).toLowerCase().includes(searchLower)
        })
      })
    }

    // Apply column filters
    Object.entries(columnFilters).forEach(([column, filterValue]) => {
      if (filterValue && filterValue.trim()) {
        const filterLower = filterValue.toLowerCase()
        filtered = filtered.filter(doc => {
          const value = doc[column]
          if (value == null) return false
          return String(value).toLowerCase().includes(filterLower)
        })
      }
    })

    return filtered
  }

  // Handler for query changes in VisualQueryBuilder
  const handleQueryChange = ({ query, options }) => {
    setCurrentQuery({ query, options })
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
      const updatedDocuments = documents.filter(doc => doc.id !== id)
      setDocuments(updatedDocuments)
      
      // Reset to first page if current page would be empty
      const newTotalPages = Math.ceil(updatedDocuments.length / documentsPerPage)
      if (currentPage > newTotalPages && newTotalPages > 0) {
        setCurrentPage(newTotalPages)
      }
      
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
    const filteredDocs = getFilteredDocuments()
    if (!filteredDocs.length) return filteredDocs

    const sorted = [...filteredDocs].sort((a, b) => {
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

    // Apply pagination
    const indexOfLastDocument = currentPage * documentsPerPage
    const indexOfFirstDocument = indexOfLastDocument - documentsPerPage
    return sorted.slice(indexOfFirstDocument, indexOfLastDocument)
  }

  const getSortIcon = (column) => {
    if (sortColumn !== column) {
      return null
    }
    return sortDirection === 'asc' ? <FiChevronUp /> : <FiChevronDown />
  }

  const handlePageChange = (pageNumber) => {
    setCurrentPage(pageNumber)
  }

  // Calculate pagination values for filtered data
  const filteredDocuments = getFilteredDocuments()
  const totalPages = Math.ceil(filteredDocuments.length / documentsPerPage)
  const indexOfLastDocument = currentPage * documentsPerPage
  const indexOfFirstDocument = indexOfLastDocument - documentsPerPage

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
    <Box position="relative" minH="100vh">
      <AnimatedBackground />
      <Box position="relative" zIndex={1} p={6}>
      <HStack justify="space-between" mb={6}>
        <VStack align="start" spacing={1}>
          <Heading 
            size="lg"
            color={useColorModeValue('gray.800', 'white')}
            bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
            px={4}
            py={2}
            borderRadius="lg"
            backdropFilter="blur(10px)"
          >
            {collection.name}
          </Heading>
          <Text color={useColorModeValue('gray.600', 'whiteAlpha.800')} ml={4}>{documents.length} documents</Text>
        </VStack>
        <IconButton
          icon={<FiRefreshCw />}
          onClick={loadCollectionData}
          variant="outline"
          aria-label="Refresh data"
        />
      </HStack>

      <Box
        bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
        borderRadius="xl"
        p={6}
        backdropFilter="blur(20px)"
        borderWidth="1px"
        borderColor={useColorModeValue('gray.200', 'whiteAlpha.200')}
        boxShadow="xl"
      >
        <Tabs index={tabIndex} onChange={handleTabChange}>
          <TabList>
            <Tab>Data</Tab>
            <Tab>Query Builder</Tab>
            <Tab>Properties</Tab>
            <Tab>Events</Tab>
            <Tab>API</Tab>
          </TabList>

          <TabPanels>
            {/* Data Tab */}
            <TabPanel>
              <VStack align="stretch" spacing={4}>
                {/* Header with search and filters */}
                <HStack justify="space-between" wrap="wrap" spacing={4}>
                  <VStack align="start" spacing={1}>
                    <Heading size="md">Documents ({filteredDocuments.length}{filteredDocuments.length !== documents.length ? ` of ${documents.length}` : ''})</Heading>
                    {filteredDocuments.length > 0 && totalPages > 1 && (
                      <Text fontSize="sm" color="gray.600">
                        Showing {indexOfFirstDocument + 1}-{Math.min(indexOfLastDocument, filteredDocuments.length)} of {filteredDocuments.length} documents
                      </Text>
                    )}
                  </VStack>
                  <HStack spacing={2}>
                    <Button
                      leftIcon={<FiPlus />}
                      colorScheme="brand"
                      onClick={handleCreateDocument}
                    >
                      Add Document
                    </Button>
                  </HStack>
                </HStack>

                {/* Search and Filter Controls */}
                <VStack align="stretch" spacing={3}>
                  <HStack spacing={3}>
                    {/* Global Search */}
                    <InputGroup flex="1" maxW="400px">
                      <InputLeftElement pointerEvents="none">
                        <FiSearch color="gray.300" />
                      </InputLeftElement>
                      <Input
                        placeholder="Search all columns..."
                        value={searchText}
                        onChange={(e) => setSearchText(e.target.value)}
                      />
                    </InputGroup>

                    {/* Advanced Filters Toggle */}
                    <Button
                      leftIcon={<FiFilter />}
                      variant={showAdvancedFilters ? "solid" : "outline"}
                      onClick={() => setShowAdvancedFilters(!showAdvancedFilters)}
                      size="sm"
                    >
                      Filters
                    </Button>

                    {/* Clear All Filters */}
                    {(searchText || Object.values(columnFilters).some(v => v)) && (
                      <Button
                        leftIcon={<FiX />}
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setSearchText('')
                          setColumnFilters({})
                        }}
                      >
                        Clear
                      </Button>
                    )}
                  </HStack>

                  {/* Column Filters */}
                  <Collapse in={showAdvancedFilters}>
                    <Box p={4} bg={useColorModeValue('gray.50', 'gray.700')} borderRadius="md">
                      <Text fontWeight="semibold" mb={3}>Filter by Column</Text>
                      <HStack wrap="wrap" spacing={3}>
                        {getTableColumns().map((column) => (
                          <Box key={column} minW="200px">
                            <Text fontSize="sm" fontWeight="medium" mb={1}>{column}</Text>
                            <Input
                              size="sm"
                              placeholder={`Filter ${column}...`}
                              value={columnFilters[column] || ''}
                              onChange={(e) => setColumnFilters(prev => ({
                                ...prev,
                                [column]: e.target.value
                              }))}
                            />
                          </Box>
                        ))}
                      </HStack>
                    </Box>
                  </Collapse>
                </VStack>

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
                                colorScheme="blue"
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

                {/* Pagination Controls */}
                {documents.length > 0 && totalPages > 1 && (
                  <HStack justify="center" spacing={2} pt={4}>
                    <IconButton
                      icon={<FiChevronLeft />}
                      size="sm"
                      variant="outline"
                      onClick={() => handlePageChange(currentPage - 1)}
                      isDisabled={currentPage === 1}
                      aria-label="Previous page"
                    />
                    
                    {Array.from({ length: totalPages }, (_, index) => (
                      <Button
                        key={index + 1}
                        size="sm"
                        variant={currentPage === index + 1 ? "solid" : "outline"}
                        colorScheme={currentPage === index + 1 ? "brand" : "gray"}
                        onClick={() => handlePageChange(index + 1)}
                      >
                        {index + 1}
                      </Button>
                    ))}
                    
                    <IconButton
                      icon={<FiChevronRight />}
                      size="sm"
                      variant="outline"
                      onClick={() => handlePageChange(currentPage + 1)}
                      isDisabled={currentPage === totalPages}
                      aria-label="Next page"
                    />
                  </HStack>
                )}

                {filteredDocuments.length === 0 && documents.length === 0 && (
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

                {filteredDocuments.length === 0 && documents.length > 0 && (
                  <Box textAlign="center" py={8}>
                    <Text color="gray.500" mb={4}>No documents match your filters</Text>
                    <Button
                      leftIcon={<FiX />}
                      variant="outline"
                      onClick={() => {
                        setSearchText('')
                        setColumnFilters({})
                      }}
                    >
                      Clear filters
                    </Button>
                  </Box>
                )}
              </VStack>
            </TabPanel>

            {/* Query Builder Tab */}
            <TabPanel>
              <VisualQueryBuilder
                collection={collection}
                schema={collection?.properties || {}}
                onQueryChange={handleQueryChange}
                onExecute={handleQueryExecute}
                initialQuery={currentQuery.query || {}}
              />
              
              {/* Query Results */}
              {queryResults.length > 0 && (
                <Box mt={6}>
                  <HStack justify="space-between" mb={4}>
                    <Heading size="md">Query Results ({queryResults.length})</Heading>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setQueryResults([])
                        setCurrentQuery({})
                      }}
                    >
                      Clear Results
                    </Button>
                  </HStack>
                  
                  <TableContainer>
                    <Table variant="simple" size="sm">
                      <Thead>
                        <Tr>
                          {getTableColumns().map((column) => (
                            <Th key={column}>{column}</Th>
                          ))}
                        </Tr>
                      </Thead>
                      <Tbody>
                        {queryResults.map((document) => (
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
                          </Tr>
                        ))}
                      </Tbody>
                    </Table>
                  </TableContainer>
                </Box>
              )}
              
              {queryLoading && (
                <Box display="flex" justifyContent="center" py={8}>
                  <VStack spacing={2}>
                    <Spinner size="lg" color="brand.500" />
                    <Text>Executing query...</Text>
                  </VStack>
                </Box>
              )}
            </TabPanel>

            {/* Properties Tab */}
            <TabPanel>
              <PropertiesEditor
                collection={collection}
                onUpdate={async (updatedCollection) => {
                  try {
                    console.log('CollectionDetail onUpdate called with:', updatedCollection)
                    console.log('Collection name:', name)
                    console.log('Properties to update:', updatedCollection.properties)
                    
                    // Update the collection via API
                    const result = await apiService.updateCollection(name, updatedCollection.properties)
                    console.log('API result:', result)
                    
                    setCollection(updatedCollection)
                    toast({
                      title: 'Collection updated',
                      description: 'Collection properties have been saved successfully.',
                      status: 'success',
                      duration: 3000,
                      isClosable: true,
                    })
                  } catch (error) {
                    console.error('Collection update error:', error)
                    toast({
                      title: 'Error updating collection',
                      description: error.response?.data?.message || error.message,
                      status: 'error',
                      duration: 5000,
                      isClosable: true,
                    })
                    throw error // Re-throw so PropertiesEditor can handle it
                  }
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
      </Box>

      {/* Document Modal */}
      <DocumentModal
        isOpen={isOpen}
        onClose={onClose}
        document={editingDocument}
        collection={collection}
        onSave={handleSaveDocument}
      />
      </Box>
    </Box>
  )
}

export default CollectionDetail