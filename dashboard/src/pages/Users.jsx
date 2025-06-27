import React, { useState, useEffect } from 'react'
import {
  Box,
  Heading,
  Button,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Badge,
  IconButton,
  VStack,
  HStack,
  Text,
  useToast,
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
  Switch,
  useDisclosure,
  Spinner,
  Center,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
} from '@chakra-ui/react'
import { FiPlus, FiEdit2, FiTrash2, FiUser } from 'react-icons/fi'
import { useAuth } from '../contexts/AuthContext'
import { AnimatedBackground } from '../components/AnimatedBackground'

function Users() {
  const [users, setUsers] = useState([])
  const [loading, setLoading] = useState(true)
  const [schemaLoading, setSchemaLoading] = useState(true)
  const [selectedUser, setSelectedUser] = useState(null)
  const [userSchema, setUserSchema] = useState(null)
  const [formData, setFormData] = useState({})
  
  const { isOpen: isCreateOpen, onOpen: onCreateOpen, onClose: onCreateClose } = useDisclosure()
  const { isOpen: isEditOpen, onOpen: onEditOpen, onClose: onEditClose } = useDisclosure()
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  
  const { masterKey } = useAuth()
  const toast = useToast()
  const cancelRef = React.useRef()

  useEffect(() => {
    fetchUserSchema()
    fetchUsers()
  }, [])

  const fetchUserSchema = async () => {
    try {
      setSchemaLoading(true)
      const response = await fetch('/_admin/collections/users', {
        headers: {
          'X-Master-Key': masterKey,
          'Content-Type': 'application/json'
        }
      })

      if (response.ok) {
        const data = await response.json()
        setUserSchema(data.properties)
        // Initialize form data with default values
        initializeFormData(data.properties)
      } else {
        // Fallback to basic schema if admin endpoint not available
        const basicSchema = {
          username: { type: 'string', required: true },
          email: { type: 'string', required: true },
          password: { type: 'string', required: true },
          role: { type: 'string', default: 'user' },
          active: { type: 'boolean', default: true }
        }
        setUserSchema(basicSchema)
        initializeFormData(basicSchema)
      }
    } catch (error) {
      console.warn('Failed to fetch user schema:', error)
      // Fallback to basic schema
      const basicSchema = {
        username: { type: 'string', required: true },
        email: { type: 'string', required: true },
        password: { type: 'string', required: true },
        role: { type: 'string', default: 'user' },
        active: { type: 'boolean', default: true }
      }
      setUserSchema(basicSchema)
      initializeFormData(basicSchema)
    } finally {
      setSchemaLoading(false)
    }
  }

  const initializeFormData = (schema) => {
    const initialData = {}
    Object.entries(schema).forEach(([field, config]) => {
      if (config.default !== undefined) {
        initialData[field] = config.default
      } else {
        switch (config.type) {
          case 'string':
            initialData[field] = ''
            break
          case 'number':
            initialData[field] = 0
            break
          case 'boolean':
            initialData[field] = false
            break
          default:
            initialData[field] = ''
        }
      }
    })
    setFormData(initialData)
  }

  const fetchUsers = async () => {
    try {
      setLoading(true)
      const response = await fetch('/users', {
        headers: {
          'X-Master-Key': masterKey,
          'Content-Type': 'application/json'
        }
      })

      if (response.ok) {
        const data = await response.json()
        setUsers(Array.isArray(data) ? data : [])
      } else {
        throw new Error('Failed to fetch users')
      }
    } catch (error) {
      toast({
        title: 'Error fetching users',
        description: error.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
      setUsers([])
    } finally {
      setLoading(false)
    }
  }

  const handleCreate = async () => {
    try {
      const response = await fetch('/users', {
        method: 'POST',
        headers: {
          'X-Master-Key': masterKey,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(formData)
      })

      if (response.ok) {
        toast({
          title: 'User created successfully',
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
        fetchUsers()
        onCreateClose()
        resetForm()
      } else {
        const error = await response.json()
        throw new Error(error.error || 'Failed to create user')
      }
    } catch (error) {
      toast({
        title: 'Error creating user',
        description: error.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const handleEdit = async () => {
    if (!selectedUser) return

    try {
      const updateData = { ...formData }
      // Don't send empty password
      if (!updateData.password) {
        delete updateData.password
      }

      const response = await fetch(`/users/${selectedUser.id}`, {
        method: 'PUT',
        headers: {
          'X-Master-Key': masterKey,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(updateData)
      })

      if (response.ok) {
        toast({
          title: 'User updated successfully',
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
        fetchUsers()
        onEditClose()
        resetForm()
      } else {
        const error = await response.json()
        throw new Error(error.error || 'Failed to update user')
      }
    } catch (error) {
      toast({
        title: 'Error updating user',
        description: error.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const handleDelete = async () => {
    if (!selectedUser) return

    try {
      const response = await fetch(`/users/${selectedUser.id}`, {
        method: 'DELETE',
        headers: {
          'X-Master-Key': masterKey,
          'Content-Type': 'application/json'
        }
      })

      if (response.ok) {
        toast({
          title: 'User deleted successfully',
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
        fetchUsers()
        onDeleteClose()
        setSelectedUser(null)
      } else {
        const error = await response.json()
        throw new Error(error.error || 'Failed to delete user')
      }
    } catch (error) {
      toast({
        title: 'Error deleting user',
        description: error.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const openCreateModal = () => {
    if (userSchema) {
      initializeFormData(userSchema)
    }
    onCreateOpen()
  }

  const openEditModal = (user) => {
    setSelectedUser(user)
    if (userSchema) {
      const editData = {}
      Object.keys(userSchema).forEach(field => {
        if (field === 'password') {
          editData[field] = '' // Don't populate password
        } else {
          editData[field] = user[field] ?? ''
        }
      })
      setFormData(editData)
    }
    onEditOpen()
  }

  const openDeleteModal = (user) => {
    setSelectedUser(user)
    onDeleteOpen()
  }

  const resetForm = () => {
    if (userSchema) {
      initializeFormData(userSchema)
    }
    setSelectedUser(null)
  }

  const formatDate = (dateStr) => {
    if (!dateStr) return 'N/A'
    try {
      return new Date(dateStr).toLocaleDateString()
    } catch {
      return 'Invalid Date'
    }
  }

  const getSortedFields = () => {
    if (!userSchema) return []
    
    // Convert to array with field names and sort by order
    return Object.entries(userSchema)
      .map(([fieldName, fieldConfig]) => ({
        name: fieldName,
        config: fieldConfig,
        order: fieldConfig.order || 999 // Default high order for fields without explicit order
      }))
      .sort((a, b) => a.order - b.order)
  }

  const renderFormField = (fieldName, fieldConfig, isCreate = true) => {
    // Skip system fields in forms, except for editable ones in create mode
    if (fieldConfig.system) {
      // For create mode, skip all system fields except the essential editable ones
      if (isCreate && !['username', 'email', 'password', 'role', 'active'].includes(fieldName)) {
        return null
      }
      // For edit mode, skip readonly system fields
      if (!isCreate && fieldConfig.readonly) {
        return null
      }
    }

    // For edit mode, password field is optional
    const isRequired = fieldConfig.required && (isCreate || fieldName !== 'password')
    
    const value = formData[fieldName] ?? ''

    switch (fieldConfig.type) {
      case 'boolean':
        return (
          <FormControl key={fieldName} display="flex" alignItems="center">
            <FormLabel mb="0">{fieldName.charAt(0).toUpperCase() + fieldName.slice(1)}</FormLabel>
            <Switch
              isChecked={value === true}
              onChange={(e) => setFormData({ ...formData, [fieldName]: e.target.checked })}
            />
          </FormControl>
        )
      
      case 'number':
        return (
          <FormControl key={fieldName} isRequired={isRequired}>
            <FormLabel>{fieldName.charAt(0).toUpperCase() + fieldName.slice(1)}</FormLabel>
            <Input
              type="number"
              value={value}
              onChange={(e) => setFormData({ ...formData, [fieldName]: parseFloat(e.target.value) || 0 })}
              placeholder={`Enter ${fieldName}`}
            />
          </FormControl>
        )
      
      case 'string':
        if (fieldName === 'password') {
          return (
            <FormControl key={fieldName} isRequired={isRequired}>
              <FormLabel>Password</FormLabel>
              <Input
                type="password"
                value={value}
                onChange={(e) => setFormData({ ...formData, [fieldName]: e.target.value })}
                placeholder={isCreate ? "Enter password" : "Leave empty to keep current password"}
              />
            </FormControl>
          )
        }
        
        if (fieldName === 'role') {
          return (
            <FormControl key={fieldName} isRequired={isRequired}>
              <FormLabel>Role</FormLabel>
              <Select
                value={value}
                onChange={(e) => setFormData({ ...formData, [fieldName]: e.target.value })}
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </Select>
            </FormControl>
          )
        }

        if (fieldName === 'email') {
          return (
            <FormControl key={fieldName} isRequired={isRequired}>
              <FormLabel>Email</FormLabel>
              <Input
                type="email"
                value={value}
                onChange={(e) => setFormData({ ...formData, [fieldName]: e.target.value })}
                placeholder="Enter email"
              />
            </FormControl>
          )
        }
        
        return (
          <FormControl key={fieldName} isRequired={isRequired}>
            <FormLabel>{fieldName.charAt(0).toUpperCase() + fieldName.slice(1)}</FormLabel>
            <Input
              value={value}
              onChange={(e) => setFormData({ ...formData, [fieldName]: e.target.value })}
              placeholder={`Enter ${fieldName}`}
            />
          </FormControl>
        )
      
      default:
        return (
          <FormControl key={fieldName} isRequired={isRequired}>
            <FormLabel>{fieldName.charAt(0).toUpperCase() + fieldName.slice(1)}</FormLabel>
            <Input
              value={value}
              onChange={(e) => setFormData({ ...formData, [fieldName]: e.target.value })}
              placeholder={`Enter ${fieldName}`}
            />
          </FormControl>
        )
    }
  }

  const getDisplayFields = () => {
    if (!userSchema) return ['username', 'email', 'role', 'active']
    
    // Show important fields first, then custom fields, skip system fields in table
    const importantFields = ['username', 'email', 'role', 'active']
    const customFields = Object.keys(userSchema).filter(
      field => !importantFields.includes(field) && 
      !['password', 'createdAt', 'updatedAt', 'id'].includes(field)
    )
    
    return [...importantFields, ...customFields]
  }

  const formatFieldValue = (value, fieldName) => {
    if (value === null || value === undefined) return 'N/A'
    
    if (userSchema && userSchema[fieldName]) {
      const fieldType = userSchema[fieldName].type
      
      if (fieldType === 'boolean') {
        return (
          <Badge colorScheme={value ? 'green' : 'red'} variant="subtle">
            {value ? 'Yes' : 'No'}
          </Badge>
        )
      }
      
      if (fieldType === 'date') {
        return formatDate(value)
      }
    }
    
    // Special handling for known fields
    if (fieldName === 'role') {
      return (
        <Badge colorScheme={value === 'admin' ? 'purple' : 'blue'} variant="subtle">
          {value}
        </Badge>
      )
    }
    
    if (fieldName === 'active') {
      return (
        <Badge colorScheme={value ? 'green' : 'red'} variant="subtle">
          {value ? 'Active' : 'Inactive'}
        </Badge>
      )
    }
    
    return String(value)
  }

  if (loading || schemaLoading) {
    return (
      <Center minH="200px">
        <VStack spacing={4}>
          <Spinner size="xl" color="brand.500" />
          <Text>Loading users...</Text>
        </VStack>
      </Center>
    )
  }

  return (
    <Box position="relative" minH="100vh">
      <AnimatedBackground />
      <Box position="relative" zIndex={1} p={6}>
      <HStack justify="space-between" mb={6}>
        <Heading size="lg" color="brand.500">
          Users Management
        </Heading>
        <Button leftIcon={<FiPlus />} colorScheme="brand" onClick={openCreateModal}>
          Add User
        </Button>
      </HStack>

      {users.length === 0 ? (
        <Box textAlign="center" py={10}>
          <FiUser size={48} style={{ margin: '0 auto 16px' }} />
          <Text fontSize="lg" color="gray.500">
            No users found
          </Text>
          <Text color="gray.400" mt={2}>
            Create your first user to get started
          </Text>
        </Box>
      ) : (
        <Box overflowX="auto">
          <Table variant="simple">
            <Thead>
              <Tr>
                {getDisplayFields().map(field => (
                  <Th key={field}>{field.charAt(0).toUpperCase() + field.slice(1)}</Th>
                ))}
                <Th>Created</Th>
                <Th>Actions</Th>
              </Tr>
            </Thead>
            <Tbody>
              {users.map((user) => (
                <Tr key={user.id}>
                  {getDisplayFields().map(field => (
                    <Td key={field} fontWeight={field === 'username' ? 'medium' : 'normal'}>
                      {formatFieldValue(user[field], field)}
                    </Td>
                  ))}
                  <Td>{formatDate(user.createdAt)}</Td>
                  <Td>
                    <HStack spacing={2}>
                      <IconButton
                        size="sm"
                        icon={<FiEdit2 />}
                        onClick={() => openEditModal(user)}
                        aria-label="Edit user"
                        variant="ghost"
                      />
                      <IconButton
                        size="sm"
                        icon={<FiTrash2 />}
                        onClick={() => openDeleteModal(user)}
                        aria-label="Delete user"
                        variant="ghost"
                        colorScheme="red"
                      />
                    </HStack>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Box>
      )}

      {/* Create User Modal */}
      <Modal isOpen={isCreateOpen} onClose={onCreateClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Create New User</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              {userSchema && getSortedFields().map(({ name: fieldName, config: fieldConfig }) => 
                renderFormField(fieldName, fieldConfig, true)
              ).filter(Boolean)}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onCreateClose}>
              Cancel
            </Button>
            <Button colorScheme="brand" onClick={handleCreate}>
              Create User
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Edit User Modal */}
      <Modal isOpen={isEditOpen} onClose={onEditClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Edit User</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4}>
              {userSchema && getSortedFields().map(({ name: fieldName, config: fieldConfig }) => 
                renderFormField(fieldName, fieldConfig, false)
              ).filter(Boolean)}
            </VStack>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onEditClose}>
              Cancel
            </Button>
            <Button colorScheme="brand" onClick={handleEdit}>
              Update User
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation Dialog */}
      <AlertDialog
        isOpen={isDeleteOpen}
        leastDestructiveRef={cancelRef}
        onClose={onDeleteClose}
      >
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              Delete User
            </AlertDialogHeader>
            <AlertDialogBody>
              Are you sure you want to delete user "{selectedUser?.username}"? 
              This action cannot be undone.
            </AlertDialogBody>
            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={onDeleteClose}>
                Cancel
              </Button>
              <Button colorScheme="red" onClick={handleDelete} ml={3}>
                Delete
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
      </Box>
    </Box>
  )
}

export default Users