import React, { useState, useEffect } from 'react'
import {
  Box,
  Grid,
  GridItem,
  Card,
  CardBody,
  CardHeader,
  Text,
  Heading,
  Button,
  IconButton,
  HStack,
  VStack,
  Badge,
  Alert,
  AlertIcon,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  InputGroup,
  InputLeftElement,
  useColorModeValue,
  Spinner,
  useToast,
  Flex,
} from '@chakra-ui/react'
import {
  FiDatabase,
  FiPlus,
  FiEdit,
  FiTrash2,
  FiEye,
  FiRefreshCw,
  FiChevronLeft,
  FiChevronRight,
  FiSearch,
} from 'react-icons/fi'
import { useNavigate } from 'react-router-dom'
import { apiService } from '../services/api'
import { AnimatedBackground } from '../components/AnimatedBackground'
import { GradientStatCard } from '../components/GradientStatCard'
import { AnimatedCard } from '../components/AnimatedCard'

function Collections() {
  const [collections, setCollections] = useState([])
  const [filteredCollections, setFilteredCollections] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [newCollectionName, setNewCollectionName] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [collectionsPerPage] = useState(12) // 4 rows x 3 columns
  const { isOpen, onOpen, onClose } = useDisclosure()
  const navigate = useNavigate()
  const toast = useToast()
  
  const cardBg = useColorModeValue('white', 'gray.700')
  const borderColor = useColorModeValue('gray.200', 'gray.600')


  useEffect(() => {
    loadCollections()
  }, [])

  // Filter collections based on search term
  useEffect(() => {
    if (!searchTerm.trim()) {
      setFilteredCollections(collections)
      setCurrentPage(1)
      return
    }

    const filtered = collections.filter(collection => 
      collection.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      Object.keys(collection.properties || {}).some(prop => 
        prop.toLowerCase().includes(searchTerm.toLowerCase())
      )
    )
    setFilteredCollections(filtered)
    setCurrentPage(1) // Reset to first page when filtering
  }, [collections, searchTerm])

  const loadCollections = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Get collections data from API
      const collectionsData = await apiService.getCollections().catch(() => [])
      
      // Use real data - no fallback needed since collections API should always work
      const collections = collectionsData || []
      
      setCollections(collections)
      setFilteredCollections(collections)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleCreateCollection = async () => {
    if (!newCollectionName.trim()) return

    try {
      // Create collection via API with basic properties
      const collectionConfig = {
        name: {
          type: 'string',
          required: true
        }
      }
      
      const newCollection = await apiService.createCollection(newCollectionName, collectionConfig)
      
      // Reload collections to get the updated list
      await loadCollections()
      
      setNewCollectionName('')
      onClose()
      
      toast({
        title: 'Collection created',
        description: `Collection "${newCollectionName}" was created successfully.`,
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Error creating collection',
        description: err.response?.data?.message || err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const handleDeleteCollection = async (name) => {
    if (!confirm(`Are you sure you want to delete the "${name}" collection?`)) return

    try {
      // Delete collection via API
      await apiService.deleteCollection(name)
      
      // Reload collections to get the updated list
      await loadCollections()
      
      toast({
        title: 'Collection deleted',
        description: `Collection "${name}" was deleted successfully.`,
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Error deleting collection',
        description: err.response?.data?.message || err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const getPropertyCount = (properties) => {
    return Object.keys(properties || {}).length
  }

  const getRequiredProperties = (properties) => {
    return Object.entries(properties || {})
      .filter(([_, prop]) => prop.required)
      .length
  }

  // Pagination logic
  const indexOfLastCollection = currentPage * collectionsPerPage
  const indexOfFirstCollection = indexOfLastCollection - collectionsPerPage
  const currentCollections = filteredCollections.slice(indexOfFirstCollection, indexOfLastCollection)
  const totalPages = Math.ceil(filteredCollections.length / collectionsPerPage)

  const handlePageChange = (pageNumber) => {
    setCurrentPage(pageNumber)
  }

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minH="300px">
        <VStack spacing={4}>
          <Spinner size="xl" color="brand.500" />
          <Text>Loading collections...</Text>
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
          onClick={loadCollections}
          size="sm"
          variant="ghost"
        />
      </Alert>
    )
  }

  return (
    <Box position="relative" minH="100vh">
      <AnimatedBackground />
      <Box position="relative" zIndex={1}>
      <VStack spacing={6} align="stretch">
        <HStack justify="space-between">
          <Heading 
            size="lg"
            color={useColorModeValue('gray.800', 'white')}
            bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
            px={4}
            py={2}
            borderRadius="lg"
            backdropFilter="blur(10px)"
          >
            Collections
          </Heading>
          <HStack spacing={2}>
            <IconButton
              icon={<FiRefreshCw />}
              onClick={loadCollections}
              variant="outline"
              aria-label="Refresh collections"
              bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
              color={useColorModeValue('gray.800', 'white')}
              borderColor={useColorModeValue('gray.300', 'whiteAlpha.300')}
              _hover={{ bg: useColorModeValue('whiteAlpha.800', 'whiteAlpha.300') }}
              backdropFilter="blur(10px)"
            />
            <Button
              leftIcon={<FiPlus />}
              colorScheme="brand"
              onClick={onOpen}
              bg="brand.500"
              _hover={{ bg: 'brand.600' }}
              boxShadow="lg"
            >
              Create Collection
            </Button>
          </HStack>
        </HStack>
        
        {/* Search Bar */}
        <Box maxW="400px">
          <InputGroup>
            <InputLeftElement pointerEvents="none">
              <FiSearch color="gray.400" />
            </InputLeftElement>
            <Input
              placeholder="Search collections or properties..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              size="md"
              bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
              borderColor={useColorModeValue('gray.300', 'whiteAlpha.300')}
              color={useColorModeValue('gray.800', 'white')}
              _placeholder={{ color: useColorModeValue('gray.500', 'whiteAlpha.600') }}
              _hover={{ borderColor: useColorModeValue('gray.400', 'whiteAlpha.400') }}
              _focus={{ 
                borderColor: 'brand.400',
                boxShadow: '0 0 0 1px var(--chakra-colors-brand-400)'
              }}
              backdropFilter="blur(10px)"
            />
          </InputGroup>
        </Box>
      </VStack>

      {filteredCollections.length > 0 && (
        <VStack spacing={6} align="stretch">
          <HStack justify="space-between" align="center">
            <Text fontSize="sm" color="gray.600">
              Showing {indexOfFirstCollection + 1}-{Math.min(indexOfLastCollection, filteredCollections.length)} of {filteredCollections.length} collections
              {searchTerm && <Text as="span" color="brand.500"> (filtered from {collections.length} total)</Text>}
            </Text>
            
            {totalPages > 1 && (
              <HStack spacing={2}>
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
          </HStack>

          <Grid templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(3, 1fr)' }} gap={6}>
            {currentCollections.map((collection) => (
          <GridItem key={collection.name}>
            <Card 
              bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
              backdropFilter="blur(20px)"
              shadow="xl"
              borderWidth="1px"
              borderColor={useColorModeValue('gray.200', 'whiteAlpha.200')}
              _hover={{ 
                shadow: '2xl',
                transform: 'translateY(-4px)',
                borderColor: 'brand.400',
                bg: useColorModeValue('whiteAlpha.950', 'blackAlpha.700')
              }}
              transition="all 0.3s"
              h="full"
            >
              <CardHeader pb={2}>
                <HStack spacing={3} align="start">
                  <Box
                    p={2}
                    borderRadius="md"
                    bg="brand.100"
                    color="brand.500"
                  >
                    <FiDatabase size={20} />
                  </Box>
                  <VStack align="start" spacing={1} flex="1">
                    <Heading size="md" color="brand.600">
                      {collection.name}
                    </Heading>
                    <HStack spacing={2}>
                      <Badge colorScheme="blue" variant="subtle">
                        {collection.documentCount} docs
                      </Badge>
                      <Badge colorScheme="green" variant="subtle">
                        {getPropertyCount(collection.properties)} fields
                      </Badge>
                    </HStack>
                  </VStack>
                </HStack>
              </CardHeader>

              <CardBody pt={2}>
                <VStack align="stretch" spacing={3}>
                  <Text fontSize="sm" color="gray.600">
                    <strong>Properties:</strong> {getPropertyCount(collection.properties)} total, {getRequiredProperties(collection.properties)} required
                  </Text>
                  
                  <Text fontSize="sm" color="gray.500">
                    <strong>Modified:</strong> {new Date(collection.lastModified).toLocaleDateString()}
                  </Text>

                  <HStack spacing={2} pt={2}>
                    <Button
                      size="sm"
                      leftIcon={<FiEye />}
                      variant="outline"
                      flex="1"
                      onClick={() => navigate(`/collections/${collection.name}`)}
                    >
                      View
                    </Button>
                    <Button
                      size="sm"
                      leftIcon={<FiEdit />}
                      variant="outline"
                      flex="1"
                      onClick={() => navigate(`/collections/${collection.name}?tab=properties`)}
                    >
                      Edit
                    </Button>
                    <IconButton
                      size="sm"
                      icon={<FiTrash2 />}
                      colorScheme="red"
                      variant="outline"
                      onClick={() => handleDeleteCollection(collection.name)}
                      aria-label="Delete collection"
                    />
                  </HStack>
                </VStack>
              </CardBody>
            </Card>
          </GridItem>
        ))}

            {/* Add new collection card only on first page */}
            {currentPage === 1 && (
              <GridItem>
                <Card 
                  bg={cardBg}
                  shadow="md"
                  borderWidth="2px"
                  borderStyle="dashed"
                  borderColor="brand.300"
                  _hover={{ 
                    borderColor: 'brand.500',
                    bg: useColorModeValue('brand.50', 'gray.600')
                  }}
                  transition="all 0.2s"
                  h="full"
                  cursor="pointer"
                  onClick={onOpen}
                >
                  <CardBody>
                    <VStack 
                      justify="center" 
                      align="center" 
                      h="full" 
                      minH="200px"
                      spacing={4}
                    >
                      <Box
                        p={4}
                        borderRadius="full"
                        bg="brand.100"
                        color="brand.500"
                      >
                        <FiPlus size={32} />
                      </Box>
                      <VStack spacing={1}>
                        <Heading size="md" color="brand.600">
                          Create Collection
                        </Heading>
                        <Text fontSize="sm" color="gray.500" textAlign="center">
                          Add a new data collection to your API
                        </Text>
                      </VStack>
                    </VStack>
                  </CardBody>
                </Card>
              </GridItem>
            )}
          </Grid>
        </VStack>
      )}

      {collections.length === 0 && !loading && (
        <VStack spacing={6} align="center" py={12}>
          <Box
            p={6}
            borderRadius="full"
            bg="gray.100"
            color="gray.400"
          >
            <FiDatabase size={48} />
          </Box>
          <VStack spacing={2}>
            <Heading size="lg" color="gray.600">
              No collections yet
            </Heading>
            <Text color="gray.500" textAlign="center">
              Create your first collection to start building your API
            </Text>
          </VStack>
          <Button
            leftIcon={<FiPlus />}
            colorScheme="brand"
            size="lg"
            onClick={onOpen}
          >
            Create Your First Collection
          </Button>
        </VStack>
      )}

      {filteredCollections.length === 0 && collections.length > 0 && !loading && (
        <VStack spacing={6} align="center" py={12}>
          <Box
            p={6}
            borderRadius="full"
            bg="gray.100"
            color="gray.400"
          >
            <FiSearch size={48} />
          </Box>
          <VStack spacing={2}>
            <Heading size="lg" color="gray.600">
              No matching collections
            </Heading>
            <Text color="gray.500" textAlign="center">
              No collections match your search term "{searchTerm}"
            </Text>
          </VStack>
          <Button
            variant="outline"
            colorScheme="brand"
            onClick={() => setSearchTerm('')}
          >
            Clear Search
          </Button>
        </VStack>
      )}

      {/* Create Collection Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Create New Collection</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl>
              <FormLabel>Collection Name</FormLabel>
              <Input
                placeholder="e.g., users, posts, products"
                value={newCollectionName}
                onChange={(e) => setNewCollectionName(e.target.value)}
                onKeyPress={(e) => {
                  if (e.key === 'Enter') {
                    handleCreateCollection()
                  }
                }}
              />
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>
              Cancel
            </Button>
            <Button 
              colorScheme="brand" 
              onClick={handleCreateCollection}
              isDisabled={!newCollectionName.trim()}
            >
              Create
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
      </Box>
    </Box>
  )
}

export default Collections