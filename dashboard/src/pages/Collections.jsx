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
  useColorModeValue,
  Spinner,
  useToast,
} from '@chakra-ui/react'
import {
  FiDatabase,
  FiPlus,
  FiEdit,
  FiTrash2,
  FiEye,
  FiRefreshCw,
} from 'react-icons/fi'
import { useNavigate } from 'react-router-dom'
import { apiService } from '../services/api'

function Collections() {
  const [collections, setCollections] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [newCollectionName, setNewCollectionName] = useState('')
  const { isOpen, onOpen, onClose } = useDisclosure()
  const navigate = useNavigate()
  const toast = useToast()
  
  const cardBg = useColorModeValue('white', 'gray.700')
  const borderColor = useColorModeValue('gray.200', 'gray.600')

  // Mock collections data for now
  const mockCollections = [
    {
      name: 'todos',
      documentCount: 3, // Updated to match actual API data
      properties: {
        title: { type: 'string', required: true },
        completed: { type: 'boolean', default: false },
        createdAt: { type: 'date', default: 'now' },
        priority: { type: 'number', default: 1 },
        description: { type: 'string' }
      },
      lastModified: new Date().toISOString()
    }
  ]

  useEffect(() => {
    loadCollections()
  }, [])

  const loadCollections = async () => {
    try {
      setLoading(true)
      setError(null)
      
      // Get real data from API
      const [collectionsData, todosData] = await Promise.all([
        apiService.getCollections().catch(() => []),
        apiService.getCollectionData('todos').catch(() => [])
      ])

      // Use real data or fallback to mock with real counts
      const collections = collectionsData.length > 0 ? collectionsData : [
        {
          ...mockCollections[0],
          documentCount: todosData.length // Use real document count
        }
      ]
      
      setCollections(collections)
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
    <Box>
      <HStack justify="space-between" mb={6}>
        <Heading size="lg">Collections</Heading>
        <HStack spacing={2}>
          <IconButton
            icon={<FiRefreshCw />}
            onClick={loadCollections}
            variant="outline"
            aria-label="Refresh collections"
          />
          <Button
            leftIcon={<FiPlus />}
            colorScheme="brand"
            onClick={onOpen}
          >
            Create Collection
          </Button>
        </HStack>
      </HStack>

      <Grid templateColumns={{ base: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(3, 1fr)' }} gap={6}>
        {collections.map((collection) => (
          <GridItem key={collection.name}>
            <Card 
              bg={cardBg}
              shadow="md"
              borderWidth="1px"
              borderColor={borderColor}
              _hover={{ 
                shadow: 'lg',
                transform: 'translateY(-2px)',
                borderColor: 'brand.300'
              }}
              transition="all 0.2s"
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

        {/* Add new collection card */}
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
      </Grid>

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
  )
}

export default Collections