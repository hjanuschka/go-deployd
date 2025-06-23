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

  const handleSaveProperty = async () => {
    if (!propertyForm.name.trim()) return

    const updatedProperties = { ...collection.properties }
    
    // If editing, remove the old property
    if (editingProperty && editingProperty !== propertyForm.name) {
      delete updatedProperties[editingProperty]
    }

    // Add/update the property
    updatedProperties[propertyForm.name] = {
      type: propertyForm.type,
      ...(propertyForm.required && { required: true }),
      ...(propertyForm.default && { default: propertyForm.default })
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
                  <Th>Name</Th>
                  <Th>Type</Th>
                  <Th>Required</Th>
                  <Th>Default</Th>
                  <Th>Actions</Th>
                </Tr>
              </Thead>
              <Tbody>
                {Object.entries(collection.properties || {}).map(([name, property]) => (
                  <Tr key={name}>
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