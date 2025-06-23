import React, { useState } from 'react'
import {
  Box,
  VStack,
  HStack,
  Text,
  Button,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
  IconButton,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  FormControl,
  FormLabel,
  Input,
  Select,
  Checkbox,
  Badge,
  FormHelperText,
  useToast,
  Card,
  CardBody,
  Heading,
} from '@chakra-ui/react'
import {
  FiPlus,
  FiEdit,
  FiTrash2,
  FiMenu,
} from 'react-icons/fi'

const PROPERTY_TYPES = [
  'string',
  'number', 
  'boolean',
  'date',
  'array',
  'object'
]

function PropertiesEditor({ collection, onUpdate }) {
  const [propertyDialogOpen, setPropertyDialogOpen] = useState(false)
  const [editingProperty, setEditingProperty] = useState(null)
  const [propertyForm, setPropertyForm] = useState({
    name: '',
    type: 'string',
    required: false,
    default: ''
  })
  const [draggedProperty, setDraggedProperty] = useState(null)
  const [draggedOverProperty, setDraggedOverProperty] = useState(null)
  const toast = useToast()

  const handleAddProperty = () => {
    setEditingProperty(null)
    setPropertyForm({
      name: '',
      type: 'string',
      required: false,
      default: ''
    })
    setPropertyDialogOpen(true)
  }

  const handleEditProperty = (name, property) => {
    setEditingProperty(name)
    setPropertyForm({
      name,
      type: property.type,
      required: property.required || false,
      default: property.default || ''
    })
    setPropertyDialogOpen(true)
  }

  const handleDeleteProperty = async (name) => {
    if (!confirm(`Are you sure you want to delete the "${name}" property?`)) return

    const updatedProperties = { ...collection.properties }
    delete updatedProperties[name]
    
    try {
      await onUpdate({
        ...collection,
        properties: updatedProperties
      })
      
      toast({
        title: 'Property deleted',
        description: `Property "${name}" was deleted successfully.`,
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Error deleting property',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const handleDragStart = (e, propertyName) => {
    setDraggedProperty(propertyName)
    e.dataTransfer.effectAllowed = 'move'
  }

  const handleDragOver = (e, propertyName) => {
    e.preventDefault()
    setDraggedOverProperty(propertyName)
    e.dataTransfer.dropEffect = 'move'
  }

  const handleDragEnd = () => {
    setDraggedProperty(null)
    setDraggedOverProperty(null)
  }

  const handleDrop = async (e, targetPropertyName) => {
    e.preventDefault()
    
    if (draggedProperty && draggedProperty !== targetPropertyName) {
      // Reorder properties
      const properties = Object.entries(collection.properties || {})
        .filter(([name]) => !collection.properties[name]?.system) // Don't reorder system properties
        .sort(([nameA, propA], [nameB, propB]) => {
          const orderA = propA.order || 0
          const orderB = propB.order || 0
          if (orderA !== orderB) {
            return orderA - orderB
          }
          return nameA.localeCompare(nameB)
        })

      const draggedIndex = properties.findIndex(([name]) => name === draggedProperty)
      const targetIndex = properties.findIndex(([name]) => name === targetPropertyName)
      
      if (draggedIndex !== -1 && targetIndex !== -1) {
        // Remove dragged item and insert at target position
        const [draggedItem] = properties.splice(draggedIndex, 1)
        properties.splice(targetIndex, 0, draggedItem)
        
        // Update order values
        const updatedProperties = { ...collection.properties }
        properties.forEach(([name, property], index) => {
          updatedProperties[name] = {
            ...property,
            order: index + 1
          }
        })
        
        try {
          await onUpdate({
            ...collection,
            properties: updatedProperties
          })
          toast({
            title: 'Properties reordered',
            status: 'success',
            duration: 2000,
            isClosable: true,
          })
        } catch (error) {
          toast({
            title: 'Error reordering properties',
            description: error.message,
            status: 'error',
            duration: 5000,
            isClosable: true,
          })
        }
      }
    }
    
    setDraggedProperty(null)
    setDraggedOverProperty(null)
  }

  const handleSaveProperty = async () => {
    if (!propertyForm.name.trim()) return

    const updatedProperties = { ...collection.properties }
    
    // If editing, remove the old property
    if (editingProperty && editingProperty !== propertyForm.name) {
      delete updatedProperties[editingProperty]
    }

    // Calculate order for new properties
    let order
    if (!editingProperty) {
      // For new properties, assign the next available order after the last moveable property
      // but before system properties (which have orders 9998, 9999)
      const moveableOrders = Object.values(updatedProperties)
        .map(prop => prop.order || 0)
        .filter(o => o > 0 && o < 9000) // Exclude system property orders
      order = moveableOrders.length > 0 ? Math.max(...moveableOrders) + 1 : 1
    } else {
      // For edited properties, keep existing order
      order = collection.properties[editingProperty]?.order || 1
    }

    // Add/update the property
    updatedProperties[propertyForm.name] = {
      type: propertyForm.type,
      ...(propertyForm.required && { required: true }),
      ...(propertyForm.default && { default: propertyForm.default }),
      order: order
    }

    try {
      await onUpdate({
        ...collection,
        properties: updatedProperties
      })
      
      toast({
        title: editingProperty ? 'Property updated' : 'Property added',
        description: `Property "${propertyForm.name}" was ${editingProperty ? 'updated' : 'added'} successfully.`,
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
      
      setPropertyDialogOpen(false)
    } catch (err) {
      toast({
        title: `Error ${editingProperty ? 'updating' : 'adding'} property`,
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const formatDefaultValue = (value, type) => {
    if (!value) return '-'
    if (value === 'now' && type === 'date') return 'Current timestamp'
    return String(value)
  }

  return (
    <Box>
      <HStack justify="space-between" mb={6}>
        <Heading size="md">Schema Properties</Heading>
        <Button
          leftIcon={<FiPlus />}
          colorScheme="brand"
          onClick={handleAddProperty}
        >
          Add Property
        </Button>
      </HStack>

      <Card>
        <CardBody p={0}>
          <TableContainer>
            <Table variant="simple">
              <Thead>
                <Tr>
                  <Th width="20px"></Th>
                  <Th>Name</Th>
                  <Th>Type</Th>
                  <Th>Required</Th>
                  <Th>Default</Th>
                  <Th>Actions</Th>
                </Tr>
              </Thead>
              <Tbody>
                {Object.entries(collection.properties || {})
                  .sort(([nameA, propA], [nameB, propB]) => {
                    // Sort by order field if present, otherwise by name
                    const orderA = propA.order || 0
                    const orderB = propB.order || 0
                    if (orderA !== orderB) {
                      return orderA - orderB
                    }
                    return nameA.localeCompare(nameB)
                  })
                  .map(([name, property]) => (
                  <Tr 
                    key={name}
                    draggable={!property.system}
                    onDragStart={(e) => handleDragStart(e, name)}
                    onDragOver={(e) => handleDragOver(e, name)}
                    onDragEnd={handleDragEnd}
                    onDrop={(e) => handleDrop(e, name)}
                    bg={draggedProperty === name ? 'blue.50' : 
                        draggedOverProperty === name ? 'gray.50' : 'transparent'}
                    cursor={property.system ? 'default' : 'move'}
                    opacity={draggedProperty === name ? 0.5 : 1}
                  >
                    <Td p={2}>
                      {!property.system && (
                        <IconButton
                          size="xs"
                          variant="ghost"
                          icon={<FiMenu />}
                          cursor="grab"
                          color="gray.400"
                          _hover={{ color: 'gray.600' }}
                          aria-label="Drag to reorder"
                        />
                      )}
                    </Td>
                    <Td>
                      <Text fontFamily="mono">{name}</Text>
                    </Td>
                    <Td>
                      <HStack spacing={2}>
                        <Badge colorScheme="blue" variant="subtle">
                          {property.type}
                        </Badge>
                        {property.readonly && (
                          <Badge colorScheme="gray" variant="solid" fontSize="xs">
                            readonly
                          </Badge>
                        )}
                        {property.system && (
                          <Badge colorScheme="orange" variant="solid" fontSize="xs">
                            system
                          </Badge>
                        )}
                      </HStack>
                    </Td>
                    <Td>
                      {property.required ? (
                        <Badge colorScheme="red">Required</Badge>
                      ) : (
                        <Badge variant="outline">Optional</Badge>
                      )}
                    </Td>
                    <Td>
                      <Text color="gray.500">
                        {formatDefaultValue(property.default, property.type)}
                      </Text>
                    </Td>
                    <Td>
                      {property.system ? (
                        <Badge colorScheme="gray" variant="subtle" fontSize="xs">
                          System managed
                        </Badge>
                      ) : (
                        <HStack spacing={1}>
                          <IconButton
                            size="sm"
                            icon={<FiEdit />}
                            variant="outline"
                            onClick={() => handleEditProperty(name, property)}
                            aria-label="Edit property"
                          />
                          <IconButton
                            size="sm"
                            icon={<FiTrash2 />}
                            colorScheme="red"
                            variant="outline"
                            onClick={() => handleDeleteProperty(name)}
                            aria-label="Delete property"
                          />
                        </HStack>
                      )}
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </TableContainer>

          {Object.keys(collection.properties || {}).length === 0 && (
            <Box textAlign="center" py={8}>
              <Text color="gray.500" mb={4}>No properties defined yet</Text>
              <Button
                leftIcon={<FiPlus />}
                colorScheme="brand"
                onClick={handleAddProperty}
              >
                Add Your First Property
              </Button>
            </Box>
          )}
        </CardBody>
      </Card>

      {/* Property Editor Modal */}
      <Modal isOpen={propertyDialogOpen} onClose={() => setPropertyDialogOpen(false)} size="md">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>
            {editingProperty ? 'Edit Property' : 'Add New Property'}
          </ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              <FormControl isRequired>
                <FormLabel>Property Name</FormLabel>
                <Input
                  value={propertyForm.name}
                  onChange={(e) => setPropertyForm({ ...propertyForm, name: e.target.value })}
                  placeholder="e.g., title, email, age"
                />
              </FormControl>

              <FormControl>
                <FormLabel>Type</FormLabel>
                <Select
                  value={propertyForm.type}
                  onChange={(e) => setPropertyForm({ ...propertyForm, type: e.target.value })}
                >
                  {PROPERTY_TYPES.map((type) => (
                    <option key={type} value={type}>
                      {type}
                    </option>
                  ))}
                </Select>
              </FormControl>

              <FormControl>
                <FormLabel>Default Value</FormLabel>
                <Input
                  value={propertyForm.default}
                  onChange={(e) => setPropertyForm({ ...propertyForm, default: e.target.value })}
                  placeholder={propertyForm.type === 'date' ? 'Use "now" for current timestamp' : 'Optional default value'}
                />
                <FormHelperText>
                  {propertyForm.type === 'date' ? 'Use "now" for current timestamp' :
                   propertyForm.type === 'boolean' ? 'Use true or false' :
                   propertyForm.type === 'number' ? 'Use numeric values like 0, 1.5, etc.' :
                   'Leave empty for no default'}
                </FormHelperText>
              </FormControl>

              <FormControl>
                <Checkbox
                  isChecked={propertyForm.required}
                  onChange={(e) => setPropertyForm({ ...propertyForm, required: e.target.checked })}
                >
                  Required field
                </Checkbox>
              </FormControl>
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={() => setPropertyDialogOpen(false)}>
              Cancel
            </Button>
            <Button 
              colorScheme="brand"
              onClick={handleSaveProperty}
              isDisabled={!propertyForm.name.trim()}
            >
              {editingProperty ? 'Update' : 'Add'} Property
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}

export default PropertiesEditor