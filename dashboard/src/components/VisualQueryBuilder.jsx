import React, { useState, useCallback, useEffect } from 'react'
import {
  Box,
  Button,
  Select,
  Input,
  VStack,
  HStack,
  Text,
  Heading,
  IconButton,
  Textarea,
  Badge,
  Card,
  CardBody,
  CardHeader,
  useToast,
  useColorModeValue
} from '@chakra-ui/react'
import { FiX, FiPlus, FiPlay, FiCopy, FiDownload } from 'react-icons/fi'

const VisualQueryBuilder = ({ 
  collection, 
  onQueryChange, 
  onExecute,
  initialQuery = {},
  schema = {},
  className = ""
}) => {
  const [conditions, setConditions] = useState([])
  const [logicalOperator, setLogicalOperator] = useState('$and')
  const [sortOptions, setSortOptions] = useState({})
  const [limitValue, setLimitValue] = useState('')
  const [skipValue, setSkipValue] = useState('')
  const [showRawQuery, setShowRawQuery] = useState(false)
  const [rawQuery, setRawQuery] = useState('')

  // Initialize from initialQuery
  useEffect(() => {
    if (initialQuery && Object.keys(initialQuery).length > 0) {
      parseQuery(initialQuery)
    }
  }, [initialQuery])

  // Operators for different field types
  const getOperatorsForType = (type) => {
    const baseOperators = [
      { value: '$eq', label: 'equals', symbol: '=' },
      { value: '$ne', label: 'not equals', symbol: '≠' },
      { value: '$exists', label: 'exists', symbol: '∃' },
    ]

    const stringOperators = [
      { value: '$regex', label: 'matches pattern', symbol: '~' },
      { value: '$in', label: 'in list', symbol: '∈' },
      { value: '$nin', label: 'not in list', symbol: '∉' },
    ]

    const numberOperators = [
      { value: '$gt', label: 'greater than', symbol: '>' },
      { value: '$gte', label: 'greater than or equal', symbol: '≥' },
      { value: '$lt', label: 'less than', symbol: '<' },
      { value: '$lte', label: 'less than or equal', symbol: '≤' },
      { value: '$in', label: 'in list', symbol: '∈' },
      { value: '$nin', label: 'not in list', symbol: '∉' },
    ]

    const dateOperators = [
      { value: '$gt', label: 'after', symbol: '>' },
      { value: '$gte', label: 'on or after', symbol: '≥' },
      { value: '$lt', label: 'before', symbol: '<' },
      { value: '$lte', label: 'on or before', symbol: '≤' },
    ]

    const arrayOperators = [
      { value: '$size', label: 'size equals', symbol: '#' },
      { value: '$elemMatch', label: 'contains element', symbol: '∋' },
    ]

    switch (type) {
      case 'string':
        return [...baseOperators, ...stringOperators]
      case 'number':
        return [...baseOperators, ...numberOperators]
      case 'date':
        return [...baseOperators, ...dateOperators]
      case 'array':
        return [...baseOperators, ...arrayOperators]
      case 'boolean':
        return baseOperators.filter(op => ['$eq', '$ne', '$exists'].includes(op.value))
      default:
        return [...baseOperators, ...stringOperators, ...numberOperators]
    }
  }

  // Get field type from schema
  const getFieldType = (fieldName) => {
    if (schema[fieldName]) {
      return schema[fieldName].type || 'string'
    }
    
    // Try to infer from field name
    if (fieldName.includes('date') || fieldName.includes('time') || fieldName.includes('At')) {
      return 'date'
    }
    if (fieldName.includes('count') || fieldName.includes('age') || fieldName.includes('price')) {
      return 'number'
    }
    if (fieldName.includes('tags') || fieldName.includes('items') || fieldName.includes('list')) {
      return 'array'
    }
    if (fieldName.includes('active') || fieldName.includes('enabled') || fieldName.includes('is')) {
      return 'boolean'
    }
    
    return 'string'
  }

  // Add a new condition
  const addCondition = useCallback(() => {
    const newCondition = {
      id: Date.now() + Math.random(),
      field: '',
      operator: '$eq',
      value: '',
      type: 'string'
    }
    setConditions(prev => [...prev, newCondition])
  }, [])

  // Remove a condition
  const removeCondition = useCallback((id) => {
    setConditions(prev => prev.filter(c => c.id !== id))
  }, [])

  // Update a condition
  const updateCondition = useCallback((id, field, value) => {
    setConditions(prev => prev.map(c => {
      if (c.id === id) {
        const updates = { [field]: value }
        
        // If field changed, update type and reset operator if needed
        if (field === 'field') {
          const newType = getFieldType(value)
          updates.type = newType
          
          // Reset operator if current one is not valid for new type
          const validOperators = getOperatorsForType(newType)
          if (!validOperators.find(op => op.value === c.operator)) {
            updates.operator = validOperators[0]?.value || '$eq'
          }
          
          // Reset value when field changes
          updates.value = ''
        }
        
        return { ...c, ...updates }
      }
      return c
    }))
  }, [getFieldType])

  // Parse a MongoDB query into visual form
  const parseQuery = (query) => {
    const newConditions = []
    let conditionId = 1

    const parseCondition = (field, value) => {
      if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
        // Field with operators
        Object.entries(value).forEach(([operator, operatorValue]) => {
          newConditions.push({
            id: conditionId++,
            field,
            operator,
            value: Array.isArray(operatorValue) ? operatorValue.join(', ') : operatorValue,
            type: getFieldType(field)
          })
        })
      } else {
        // Simple equality
        newConditions.push({
          id: conditionId++,
          field,
          operator: '$eq',
          value: Array.isArray(value) ? value.join(', ') : value,
          type: getFieldType(field)
        })
      }
    }

    Object.entries(query).forEach(([key, value]) => {
      if (key === '$or' || key === '$and') {
        setLogicalOperator(key)
        if (Array.isArray(value)) {
          value.forEach(condition => {
            Object.entries(condition).forEach(([field, fieldValue]) => {
              parseCondition(field, fieldValue)
            })
          })
        }
      } else if (!key.startsWith('$')) {
        parseCondition(key, value)
      }
    })

    if (newConditions.length > 0) {
      setConditions(newConditions)
    }
  }

  // Build MongoDB query from visual conditions
  const buildQuery = useCallback(() => {
    const mongoQuery = {}
    
    if (conditions.length === 0) {
      return mongoQuery
    }

    const queryConditions = []

    conditions.forEach(condition => {
      if (!condition.field || condition.value === '') return

      let value = condition.value
      
      // Parse value based on operator and type
      if (condition.operator === '$in' || condition.operator === '$nin') {
        // Parse comma-separated values
        value = value.toString().split(',').map(v => v.trim()).filter(v => v)
        
        // Convert to appropriate types
        if (condition.type === 'number') {
          value = value.map(v => {
            const num = parseFloat(v)
            return isNaN(num) ? v : num
          })
        }
      } else if (condition.type === 'number' && !isNaN(value)) {
        value = parseFloat(value)
      } else if (condition.type === 'boolean') {
        value = value === 'true' || value === true
      } else if (condition.operator === '$exists') {
        value = value === 'true' || value === true
      } else if (condition.operator === '$size') {
        value = parseInt(value) || 0
      }

      const conditionQuery = {}
      
      if (condition.operator === '$eq') {
        conditionQuery[condition.field] = value
      } else {
        conditionQuery[condition.field] = { [condition.operator]: value }
      }

      queryConditions.push(conditionQuery)
    })

    // Combine conditions based on logical operator
    if (queryConditions.length === 1) {
      Object.assign(mongoQuery, queryConditions[0])
    } else if (queryConditions.length > 1) {
      mongoQuery[logicalOperator] = queryConditions
    }

    return mongoQuery
  }, [conditions, logicalOperator])

  // Build query options
  const buildOptions = useCallback(() => {
    const options = {}
    
    if (Object.keys(sortOptions).length > 0) {
      options.$sort = sortOptions
    }
    
    if (limitValue && !isNaN(limitValue)) {
      options.$limit = parseInt(limitValue)
    }
    
    if (skipValue && !isNaN(skipValue)) {
      options.$skip = parseInt(skipValue)
    }
    
    return options
  }, [sortOptions, limitValue, skipValue])

  // Handle query change
  useEffect(() => {
    const query = buildQuery()
    const options = buildOptions()
    onQueryChange?.({ query, options })
  }, [buildQuery, buildOptions, onQueryChange])

  // Execute query
  const handleExecute = () => {
    const query = buildQuery()
    const options = buildOptions()
    onExecute?.({ query, options })
  }

  // Export query
  const handleExport = async () => {
    const query = buildQuery()
    const options = buildOptions()
    const fullQuery = { ...query, ...options }
    
    try {
      await navigator.clipboard.writeText(JSON.stringify(fullQuery, null, 2))
      toast({
        title: 'Query copied to clipboard!',
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Failed to copy query',
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  // Import query
  const handleImport = () => {
    try {
      const parsed = JSON.parse(rawQuery)
      parseQuery(parsed)
      setShowRawQuery(false)
      toast({
        title: 'Query imported successfully!',
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Invalid JSON query',
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  // Add sort option
  const addSort = () => {
    setSortOptions(prev => ({ ...prev, '': 1 }))
  }

  // Update sort option
  const updateSort = (oldField, newField, direction) => {
    setSortOptions(prev => {
      const newSort = { ...prev }
      if (oldField !== newField) {
        delete newSort[oldField]
      }
      newSort[newField] = direction
      return newSort
    })
  }

  // Remove sort option
  const removeSort = (field) => {
    setSortOptions(prev => {
      const newSort = { ...prev }
      delete newSort[field]
      return newSort
    })
  }

  // Get available fields from schema or common fields
  const getAvailableFields = () => {
    if (Object.keys(schema).length > 0) {
      return Object.keys(schema)
    }
    
    // Common fields for any collection
    return [
      'id', 'name', 'title', 'email', 'status', 'type', 'category',
      'createdAt', 'updatedAt', 'userId', 'active', 'enabled',
      'priority', 'order', 'count', 'amount', 'price', 'tags'
    ]
  }

  const availableFields = getAvailableFields()

  return (
    <Card bg={useColorModeValue('white', 'gray.800')} className={className}>
      <CardHeader>
        <HStack justify="space-between">
          <Heading size="md">Query Builder</Heading>
          <HStack spacing={2}>
            <Button
              size="sm"
              variant="outline"
              onClick={() => setShowRawQuery(!showRawQuery)}
            >
              {showRawQuery ? 'Visual' : 'Raw'}
            </Button>
            <Button
              size="sm"
              colorScheme="blue"
              leftIcon={<FiCopy />}
              onClick={handleExport}
            >
              Export
            </Button>
            <Button
              size="sm"
              colorScheme="green"
              leftIcon={<FiPlay />}
              onClick={handleExecute}
            >
              Execute
            </Button>
          </HStack>
        </HStack>
      </CardHeader>
      <CardBody>

        {showRawQuery ? (
          /* Raw Query Editor */
          <VStack spacing={4} align="stretch">
            <Box>
              <Text fontSize="sm" fontWeight="medium" mb={2}>
                MongoDB Query (JSON)
              </Text>
              <Textarea
                value={rawQuery}
                onChange={(e) => setRawQuery(e.target.value)}
                h="64"
                fontFamily="mono"
                fontSize="sm"
                resize="none"
                placeholder={`{
  "status": "active",
  "age": { "$gte": 18 },
  "$or": [
    { "role": "admin" },
    { "permissions": { "$in": ["write", "delete"] } }
  ]
}`}
              />
            </Box>
            <HStack justify="flex-end">
              <Button
                colorScheme="blue"
                onClick={handleImport}
              >
                Import Query
              </Button>
            </HStack>
          </VStack>
        ) : (
          /* Visual Query Builder */
          <VStack spacing={6} align="stretch">
            {/* Logical Operator */}
            {conditions.length > 1 && (
              <HStack spacing={2}>
                <Text fontSize="sm">Combine conditions with:</Text>
                <Select
                  value={logicalOperator}
                  onChange={(e) => setLogicalOperator(e.target.value)}
                  size="sm"
                  w="auto"
                >
                  <option value="$and">AND</option>
                  <option value="$or">OR</option>
                </Select>
              </HStack>
            )}

            {/* Conditions */}
            <VStack spacing={3} align="stretch">
              {conditions.map((condition, index) => (
                <Box key={condition.id} p={3} bg={useColorModeValue('gray.50', 'gray.700')} borderRadius="lg">
                  <HStack spacing={2} align="center">
                    {index > 0 && (
                      <Badge colorScheme="blue" fontSize="xs" minW="30px">
                        {logicalOperator === '$and' ? 'AND' : 'OR'}
                      </Badge>
                    )}
                    
                    {/* Field */}
                    <Select
                      value={condition.field}
                      onChange={(e) => updateCondition(condition.id, 'field', e.target.value)}
                      size="sm"
                      flex="1"
                    >
                      <option value="">Select field...</option>
                      {availableFields.map(field => (
                        <option key={field} value={field}>{field}</option>
                      ))}
                    </Select>

                    {/* Operator */}
                    <Select
                      value={condition.operator}
                      onChange={(e) => updateCondition(condition.id, 'operator', e.target.value)}
                      size="sm"
                      isDisabled={!condition.field}
                      w="auto"
                    >
                      {getOperatorsForType(condition.type).map(op => (
                        <option key={op.value} value={op.value}>
                          {op.symbol} {op.label}
                        </option>
                      ))}
                    </Select>

                    {/* Value */}
                    {condition.operator === '$exists' ? (
                      <Select
                        value={condition.value}
                        onChange={(e) => updateCondition(condition.id, 'value', e.target.value)}
                        size="sm"
                        flex="1"
                      >
                        <option value="true">true</option>
                        <option value="false">false</option>
                      </Select>
                    ) : condition.type === 'boolean' ? (
                      <Select
                        value={condition.value}
                        onChange={(e) => updateCondition(condition.id, 'value', e.target.value)}
                        size="sm"
                        flex="1"
                      >
                        <option value="">Select...</option>
                        <option value="true">true</option>
                        <option value="false">false</option>
                      </Select>
                    ) : condition.type === 'date' ? (
                      <Input
                        type="datetime-local"
                        value={condition.value}
                        onChange={(e) => updateCondition(condition.id, 'value', e.target.value)}
                        size="sm"
                        flex="1"
                      />
                    ) : (
                      <Input
                        type={condition.type === 'number' ? 'number' : 'text'}
                        value={condition.value}
                        onChange={(e) => updateCondition(condition.id, 'value', e.target.value)}
                        placeholder={
                          condition.operator === '$in' || condition.operator === '$nin' 
                            ? 'value1, value2, value3' 
                            : 'Enter value...'
                        }
                        size="sm"
                        flex="1"
                      />
                    )}

                    {/* Remove button */}
                    <IconButton
                      onClick={() => removeCondition(condition.id)}
                      icon={<FiX />}
                      size="sm"
                      colorScheme="red"
                      variant="ghost"
                      aria-label="Remove condition"
                    />
                  </HStack>
                </Box>
              ))}

              {/* Add Condition Button */}
              <Button
                onClick={addCondition}
                variant="outline"
                borderStyle="dashed"
                w="full"
                leftIcon={<FiPlus />}
              >
                Add Condition
              </Button>
            </VStack>

            {/* Sort Options */}
            <VStack spacing={3} align="stretch">
              <Text fontSize="sm" fontWeight="medium">Sort Options</Text>
              {Object.entries(sortOptions).map(([field, direction]) => (
                <HStack key={field || 'new'} spacing={2}>
                  <Select
                    value={field}
                    onChange={(e) => updateSort(field, e.target.value, direction)}
                    size="sm"
                    flex="1"
                  >
                    <option value="">Select field...</option>
                    {availableFields.map(f => (
                      <option key={f} value={f}>{f}</option>
                    ))}
                  </Select>
                  <Select
                    value={direction}
                    onChange={(e) => updateSort(field, field, parseInt(e.target.value))}
                    size="sm"
                    w="auto"
                  >
                    <option value={1}>Ascending</option>
                    <option value={-1}>Descending</option>
                  </Select>
                  <IconButton
                    onClick={() => removeSort(field)}
                    icon={<FiX />}
                    size="sm"
                    colorScheme="red"
                    variant="ghost"
                    aria-label="Remove sort"
                  />
                </HStack>
              ))}
              <Button
                onClick={addSort}
                variant="link"
                size="sm"
                alignSelf="flex-start"
                leftIcon={<FiPlus />}
              >
                Add Sort Field
              </Button>
            </VStack>

            {/* Limit and Skip */}
            <HStack spacing={4}>
              <Box flex="1">
                <Text fontSize="sm" fontWeight="medium" mb={2}>Limit</Text>
                <Input
                  type="number"
                  value={limitValue}
                  onChange={(e) => setLimitValue(e.target.value)}
                  placeholder="Max records"
                  min="1"
                  size="sm"
                />
              </Box>
              <Box flex="1">
                <Text fontSize="sm" fontWeight="medium" mb={2}>Skip</Text>
                <Input
                  type="number"
                  value={skipValue}
                  onChange={(e) => setSkipValue(e.target.value)}
                  placeholder="Records to skip"
                  min="0"
                  size="sm"
                />
              </Box>
            </HStack>

            {/* Query Preview */}
            <Box p={3} bg={useColorModeValue('gray.100', 'gray.900')} borderRadius="lg" borderWidth="1px">
              <Text fontSize="xs" color="gray.500" mb={2}>Generated Query:</Text>
              <Text as="pre" fontSize="xs" fontFamily="mono" overflowX="auto" whiteSpace="pre-wrap">
                {JSON.stringify({ ...buildQuery(), ...buildOptions() }, null, 2)}
              </Text>
            </Box>
          </VStack>
        )}
      </CardBody>
    </Card>
  )
}

export default VisualQueryBuilder