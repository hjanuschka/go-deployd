import React, { useState, useEffect } from 'react'
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  Button,
  VStack,
  FormControl,
  FormLabel,
  Input,
  Switch,
  Textarea,
  NumberInput,
  NumberInputField,
  Alert,
  AlertIcon,
  Text,
  Badge,
} from '@chakra-ui/react'

function DocumentModal({ isOpen, onClose, document, collection, onSave }) {
  const [formData, setFormData] = useState({})
  const [errors, setErrors] = useState({})

  useEffect(() => {
    if (isOpen) {
      if (document) {
        // Editing existing document
        setFormData({ ...document })
      } else {
        // Creating new document
        const initialData = {}
        Object.entries(collection.properties || {}).forEach(([name, property]) => {
          if (property.default !== undefined) {
            if (property.default === 'now' && property.type === 'date') {
              initialData[name] = new Date().toISOString().split('T')[0]
            } else {
              initialData[name] = property.default
            }
          }
        })
        setFormData(initialData)
      }
      setErrors({})
    }
  }, [isOpen, document, collection])

  const handleFieldChange = (name, value) => {
    setFormData({ ...formData, [name]: value })
    
    // Clear error for this field
    if (errors[name]) {
      setErrors({ ...errors, [name]: null })
    }
  }

  const validateForm = () => {
    const newErrors = {}
    
    Object.entries(collection.properties || {}).forEach(([name, property]) => {
      if (property.required && (!formData[name] || formData[name] === '')) {
        newErrors[name] = 'This field is required'
      }
    })

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSave = () => {
    if (!validateForm()) return

    // Process the data before saving
    const processedData = { ...formData }
    
    // Convert date strings to ISO strings
    Object.entries(collection.properties || {}).forEach(([name, property]) => {
      if (property.type === 'date' && processedData[name]) {
        if (typeof processedData[name] === 'string' && !processedData[name].includes('T')) {
          // Convert date input to ISO string
          processedData[name] = new Date(processedData[name]).toISOString()
        }
      }
    })

    onSave(processedData)
  }

  const renderField = (name, property) => {
    const value = formData[name] || ''
    const error = errors[name]

    switch (property.type) {
      case 'string':
        if (name === 'description' || name.includes('text') || name.includes('content')) {
          return (
            <Textarea
              value={value}
              onChange={(e) => handleFieldChange(name, e.target.value)}
              placeholder={`Enter ${name}...`}
              isInvalid={!!error}
            />
          )
        }
        return (
          <Input
            value={value}
            onChange={(e) => handleFieldChange(name, e.target.value)}
            placeholder={`Enter ${name}...`}
            isInvalid={!!error}
          />
        )

      case 'number':
        return (
          <NumberInput
            value={value}
            onChange={(valueString, valueNumber) => handleFieldChange(name, valueNumber || 0)}
            isInvalid={!!error}
          >
            <NumberInputField placeholder={`Enter ${name}...`} />
          </NumberInput>
        )

      case 'boolean':
        return (
          <Switch
            isChecked={!!value}
            onChange={(e) => handleFieldChange(name, e.target.checked)}
          />
        )

      case 'date':
        return (
          <Input
            type="date"
            value={value ? new Date(value).toISOString().split('T')[0] : ''}
            onChange={(e) => handleFieldChange(name, e.target.value)}
            isInvalid={!!error}
          />
        )

      case 'array':
        return (
          <Textarea
            value={Array.isArray(value) ? value.join(', ') : value}
            onChange={(e) => handleFieldChange(name, e.target.value.split(',').map(v => v.trim()))}
            placeholder="Comma-separated values"
            isInvalid={!!error}
          />
        )

      default:
        return (
          <Textarea
            value={typeof value === 'object' ? JSON.stringify(value, null, 2) : value}
            onChange={(e) => {
              try {
                const parsed = JSON.parse(e.target.value)
                handleFieldChange(name, parsed)
              } catch {
                handleFieldChange(name, e.target.value)
              }
            }}
            placeholder="JSON object"
            isInvalid={!!error}
          />
        )
    }
  }

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="lg">
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>
          {document ? 'Edit Document' : 'Create New Document'}
        </ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          <VStack spacing={4}>
            {document && (
              <Alert status="info">
                <AlertIcon />
                <Text fontSize="sm">Document ID: {document.id}</Text>
              </Alert>
            )}
            
            {Object.entries(collection.properties || {}).map(([name, property]) => (
              <FormControl key={name} isRequired={property.required} isInvalid={!!errors[name]}>
                <FormLabel>
                  {name}
                  <Badge ml={2} colorScheme="blue" variant="subtle" fontSize="xs">
                    {property.type}
                  </Badge>
                  {property.required && (
                    <Badge ml={1} colorScheme="red" variant="subtle" fontSize="xs">
                      required
                    </Badge>
                  )}
                </FormLabel>
                {renderField(name, property)}
                {errors[name] && (
                  <Text color="red.500" fontSize="sm" mt={1}>
                    {errors[name]}
                  </Text>
                )}
              </FormControl>
            ))}

            {Object.keys(collection.properties || {}).length === 0 && (
              <Alert status="warning">
                <AlertIcon />
                <Text>No properties defined for this collection. Add properties in the Properties tab first.</Text>
              </Alert>
            )}
          </VStack>
        </ModalBody>
        <ModalFooter>
          <Button variant="ghost" mr={3} onClick={onClose}>
            Cancel
          </Button>
          <Button 
            colorScheme="brand" 
            onClick={handleSave}
            isDisabled={Object.keys(collection.properties || {}).length === 0}
          >
            {document ? 'Update' : 'Create'}
          </Button>
        </ModalFooter>
      </ModalContent>
    </Modal>
  )
}

export default DocumentModal