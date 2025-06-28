import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react'
import {
  Box,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Input,
  Select,
  Checkbox,
  IconButton,
  HStack,
  VStack,
  Text,
  Button,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  useColorModeValue,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  Badge,
  Tooltip,
  InputGroup,
  InputLeftElement,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
} from '@chakra-ui/react'
import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  getPaginationRowModel,
  useReactTable,
} from '@tanstack/react-table'
import {
  FiChevronDown,
  FiChevronUp,
  FiFilter,
  FiTrash2,
  FiCopy,
  FiClipboard,
  FiSave,
  FiPlus,
  FiSearch,
  FiMoreVertical,
  FiDownload,
  FiUpload,
} from 'react-icons/fi'
import { format, parseISO } from 'date-fns'
import { apiService } from '../services/api'
import { useToast } from './ToastSystem'
import { AnimatedCard } from './AnimatedCard'
import { parseCSV, exportToCSV, getNextCell, validateData, formatCellValue } from '../utils/spreadsheetHelpers'
import { useWebSocket } from '../hooks/useWebSocket'

// Cell editor component for inline editing
const CellEditor = ({ value: initialValue, row, column, table }) => {
  const [value, setValue] = useState(initialValue)
  const inputRef = useRef(null)
  const { updateData } = table.options.meta

  useEffect(() => {
    setValue(initialValue)
  }, [initialValue])

  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [])

  const onBlur = () => {
    updateData(row.index, column.id, value)
  }

  const onKeyDown = (e) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      onBlur()
      // Move to next row
      const nextRowIndex = row.index + 1
      if (nextRowIndex < table.getRowModel().rows.length) {
        table.options.meta.setEditingCell({
          rowIndex: nextRowIndex,
          columnId: column.id
        })
      }
    } else if (e.key === 'Tab') {
      e.preventDefault()
      onBlur()
      // Move to next column
      const visibleColumns = table.getVisibleFlatColumns()
      const currentIndex = visibleColumns.findIndex(col => col.id === column.id)
      const nextColumn = visibleColumns[currentIndex + (e.shiftKey ? -1 : 1)]
      if (nextColumn) {
        table.options.meta.setEditingCell({
          rowIndex: row.index,
          columnId: nextColumn.id
        })
      }
    } else if (e.key === 'Escape') {
      setValue(initialValue)
      table.options.meta.setEditingCell(null)
    }
  }

  // Render different input types based on column data type
  const renderEditor = () => {
    const dataType = column.columnDef.meta?.dataType || 'string'

    switch (dataType) {
      case 'boolean':
        return (
          <Checkbox
            isChecked={value}
            onChange={(e) => {
              setValue(e.target.checked)
              updateData(row.index, column.id, e.target.checked)
            }}
            size="sm"
          />
        )
      case 'number':
        return (
          <Input
            ref={inputRef}
            type="number"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onBlur={onBlur}
            onKeyDown={onKeyDown}
            size="sm"
            variant="unstyled"
            px={2}
          />
        )
      case 'date':
        return (
          <Input
            ref={inputRef}
            type="date"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onBlur={onBlur}
            onKeyDown={onKeyDown}
            size="sm"
            variant="unstyled"
            px={2}
          />
        )
      case 'select':
        return (
          <Select
            ref={inputRef}
            value={value}
            onChange={(e) => {
              setValue(e.target.value)
              updateData(row.index, column.id, e.target.value)
            }}
            onKeyDown={onKeyDown}
            size="sm"
            variant="unstyled"
            px={2}
          >
            {column.columnDef.meta?.options?.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </Select>
        )
      default:
        return (
          <Input
            ref={inputRef}
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onBlur={onBlur}
            onKeyDown={onKeyDown}
            size="sm"
            variant="unstyled"
            px={2}
          />
        )
    }
  }

  return renderEditor()
}

// Main SpreadsheetEditor component
export const SpreadsheetEditor = ({
  data: propsData,
  schema,
  onUpdate,
  onRefresh,
  loading: propsLoading,
  collection,
  columns: userColumns,
  onDataChange,
  enableRealtime = true,
  pageSize = 50,
}) => {
  const [data, setData] = useState(propsData || [])
  const [loading, setLoading] = useState(propsLoading || true)
  const [error, setError] = useState(null)
  const [selectedRows, setSelectedRows] = useState({})
  const [editingCell, setEditingCell] = useState(null)
  const [columnFilters, setColumnFilters] = useState([])
  const [sorting, setSorting] = useState([])
  const [globalFilter, setGlobalFilter] = useState('')
  const toast = useToast()
  const bgColor = useColorModeValue('white', 'gray.800')
  const borderColor = useColorModeValue('gray.200', 'gray.600')
  const hoverBg = useColorModeValue('gray.50', 'gray.700')
  const selectedBg = useColorModeValue('blue.50', 'blue.900')
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const fileInputRef = useRef(null)

  // Define columns with proper configuration
  const columns = useMemo(() => {
    if (userColumns) {
      return userColumns.map(col => ({
        ...col,
        cell: ({ getValue, row, column, table }) => {
          const isEditing = editingCell?.rowIndex === row.index && 
                          editingCell?.columnId === column.id
          
          if (isEditing) {
            return (
              <CellEditor
                value={getValue()}
                row={row}
                column={column}
                table={table}
              />
            )
          }

          const dataType = column.columnDef.meta?.dataType || 'string'
          const value = getValue()

          // Format display based on data type
          const displayValue = () => {
            if (value === null || value === undefined) return '-'
            
            switch (dataType) {
              case 'boolean':
                return <Checkbox isChecked={value} isReadOnly size="sm" />
              case 'date':
                try {
                  return format(parseISO(value), 'MMM dd, yyyy')
                } catch {
                  return value
                }
              case 'number':
                return typeof value === 'number' ? value.toLocaleString() : value
              default:
                return value
            }
          }

          return (
            <Box
              onClick={() => setEditingCell({ rowIndex: row.index, columnId: column.id })}
              cursor="pointer"
              p={2}
              _hover={{ bg: hoverBg }}
              minH="32px"
            >
              {displayValue()}
            </Box>
          )
        }
      }))
    }

    // Auto-generate columns from schema or data
    if (schema && Object.keys(schema).length > 0) {
      return Object.entries(schema).map(([key, config]) => ({
        accessorKey: key,
        header: key.charAt(0).toUpperCase() + key.slice(1).replace(/_/g, ' '),
        meta: {
          dataType: config.type || 'string',
          required: config.required || false
        }
      }))
    }

    // Auto-generate columns from data if not provided
    if (data.length > 0) {
      return Object.keys(data[0]).map(key => ({
        accessorKey: key,
        header: key.charAt(0).toUpperCase() + key.slice(1).replace(/_/g, ' '),
        meta: {
          dataType: typeof data[0][key] === 'boolean' ? 'boolean' :
                   typeof data[0][key] === 'number' ? 'number' :
                   key.includes('date') || key.includes('_at') ? 'date' : 'string'
        }
      }))
    }

    return []
  }, [userColumns, schema, data, editingCell, hoverBg])

  // Table instance
  const table = useReactTable({
    data,
    columns,
    state: {
      columnFilters,
      globalFilter,
      sorting,
      rowSelection: selectedRows,
    },
    enableRowSelection: true,
    onRowSelectionChange: setSelectedRows,
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    meta: {
      updateData: (rowIndex, columnId, value) => {
        setData((old) =>
          old.map((row, index) => {
            if (index === rowIndex) {
              const updatedRow = {
                ...old[rowIndex],
                [columnId]: value,
              }
              // Save to backend
              saveRow(updatedRow)
              return updatedRow
            }
            return row
          })
        )
      },
      setEditingCell,
    },
  })

  // Load data
  const loadData = async () => {
    try {
      setLoading(true)
      setError(null)
      const response = await apiService.getCollectionData(collection)
      setData(response.data || [])
    } catch (err) {
      setError(err.message)
      toast.error('Failed to load data')
    } finally {
      setLoading(false)
    }
  }

  // Save row to backend
  const saveRow = async (row) => {
    try {
      if (onUpdate) {
        const success = await onUpdate(row.id, null, row)
        if (success) {
          toast.success('Row saved successfully')
          if (onDataChange) onDataChange()
        } else {
          toast.error('Failed to save row')
          // Reload data to sync with server
          if (onRefresh) onRefresh()
        }
      } else {
        // Fallback to direct API calls
        if (row.id) {
          await apiService.updateDocument(collection, row.id, row)
        } else {
          const response = await apiService.createDocument(collection, row)
          // Update local data with new ID
          setData(old => old.map(r => r === row ? response : r))
        }
        toast.success('Row saved successfully')
        if (onDataChange) onDataChange()
      }
    } catch (err) {
      toast.error('Failed to save row: ' + err.message)
      // Reload data to sync with server
      if (onRefresh) {
        onRefresh()
      } else {
        loadData()
      }
    }
  }

  // Delete selected rows
  const deleteSelectedRows = async () => {
    const selectedRowIds = Object.keys(selectedRows)
      .filter(id => selectedRows[id])
      .map(id => data[id].id)
      .filter(Boolean)

    try {
      await Promise.all(
        selectedRowIds.map(id => apiService.deleteDocument(collection, id))
      )
      toast.success(`Deleted ${selectedRowIds.length} rows`)
      setSelectedRows({})
      loadData()
      if (onDataChange) onDataChange()
    } catch (err) {
      toast.error('Failed to delete rows: ' + err.message)
    }
    onDeleteClose()
  }

  // Add new row
  const addNewRow = () => {
    const newRow = columns.reduce((acc, col) => {
      if (col.accessorKey !== 'id') {
        acc[col.accessorKey] = col.meta?.dataType === 'boolean' ? false : ''
      }
      return acc
    }, {})
    
    setData([newRow, ...data])
    setEditingCell({ rowIndex: 0, columnId: columns[0].accessorKey })
  }

  // Copy/paste handlers
  const handleCopy = () => {
    const selectedRowsData = Object.keys(selectedRows)
      .filter(id => selectedRows[id])
      .map(id => data[id])

    if (selectedRowsData.length > 0) {
      const text = selectedRowsData
        .map(row => columns.map(col => row[col.accessorKey]).join('\t'))
        .join('\n')
      navigator.clipboard.writeText(text)
      toast.success('Copied to clipboard')
    }
  }

  const handlePaste = async () => {
    try {
      const text = await navigator.clipboard.readText()
      const rows = text.split('\n').map(row => {
        const values = row.split('\t')
        return columns.reduce((acc, col, index) => {
          if (values[index] !== undefined) {
            acc[col.accessorKey] = values[index]
          }
          return acc
        }, {})
      })
      
      setData([...data, ...rows])
      toast.success('Pasted from clipboard')
    } catch (err) {
      toast.error('Failed to paste: ' + err.message)
    }
  }

  // Export data
  const exportData = () => {
    const filename = `${collection}_export_${new Date().toISOString().split('T')[0]}.csv`
    exportToCSV(data, columns, filename)
    toast.success('Data exported successfully')
  }

  // Import data from CSV
  const importData = async (event) => {
    const file = event.target.files[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = async (e) => {
      try {
        const text = e.target.result
        const importedData = parseCSV(text, columns)
        
        // Validate imported data
        const errors = validateData(importedData, columns)
        if (errors.length > 0) {
          toast.error(`Import failed: ${errors[0].message}`)
          return
        }
        
        // Save all imported rows
        const promises = importedData.map(row => 
          apiService.createDocument(collection, row)
        )
        
        await Promise.all(promises)
        toast.success(`Imported ${importedData.length} rows successfully`)
        loadData()
      } catch (err) {
        toast.error('Failed to import CSV: ' + err.message)
      }
    }
    
    reader.readAsText(file)
    event.target.value = '' // Reset file input
  }

  // WebSocket connection for real-time updates
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/_ws/collections/${collection}`
  
  const { isConnected } = useWebSocket(wsUrl, {
    onMessage: (message) => {
      if (!enableRealtime) return
      
      switch (message.type) {
        case 'create':
          setData(prev => [message.data, ...prev])
          toast.info('New row added')
          break
        case 'update':
          setData(prev => prev.map(item => 
            item.id === message.data.id ? message.data : item
          ))
          break
        case 'delete':
          setData(prev => prev.filter(item => item.id !== message.id))
          toast.info('Row deleted')
          break
      }
    },
    onError: (error) => {
      console.error('WebSocket error:', error)
    },
    reconnect: enableRealtime,
  })

  // Update data when props change
  useEffect(() => {
    if (propsData) {
      setData(propsData)
      setLoading(false)
    } else if (collection) {
      loadData()
    }
  }, [propsData, collection])

  // Update loading state when props change
  useEffect(() => {
    if (propsLoading !== undefined) {
      setLoading(propsLoading)
    }
  }, [propsLoading])

  // Keyboard navigation
  useEffect(() => {
    const handleKeyDown = (e) => {
      // Handle keyboard navigation when not editing
      if (!editingCell && table.getRowModel().rows.length > 0) {
        let nextCell = null
        
        // If no cell is selected, start from the first cell
        const currentCell = editingCell || { rowIndex: 0, columnId: columns[0]?.accessorKey }
        
        switch (e.key) {
          case 'ArrowDown':
          case 'ArrowUp':
          case 'ArrowLeft':
          case 'ArrowRight':
            e.preventDefault()
            nextCell = getNextCell(
              currentCell.rowIndex,
              currentCell.columnId,
              e.key,
              table
            )
            if (nextCell) {
              setEditingCell(nextCell)
            }
            break
            
          case 'Enter':
            e.preventDefault()
            setEditingCell(currentCell)
            break
            
          case 'Delete':
            if (Object.keys(selectedRows).filter(id => selectedRows[id]).length > 0) {
              e.preventDefault()
              onDeleteOpen()
            }
            break
        }
      }
      
      // Global shortcuts
      if (e.ctrlKey || e.metaKey) {
        switch (e.key) {
          case 'c':
            if (Object.keys(selectedRows).filter(id => selectedRows[id]).length > 0) {
              e.preventDefault()
              handleCopy()
            }
            break
            
          case 'v':
            e.preventDefault()
            handlePaste()
            break
            
          case 's':
            e.preventDefault()
            toast.success('All changes are automatically saved')
            break
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [editingCell, selectedRows, table, columns])

  if (loading) {
    return (
      <AnimatedCard p={8}>
        <VStack spacing={4}>
          <Spinner size="xl" color="blue.500" />
          <Text>Loading data...</Text>
        </VStack>
      </AnimatedCard>
    )
  }

  if (error) {
    return (
      <AnimatedCard p={8}>
        <Alert status="error">
          <AlertIcon />
          <AlertTitle>Error loading data</AlertTitle>
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      </AnimatedCard>
    )
  }

  return (
    <AnimatedCard p={0} overflow="hidden">
      {/* Toolbar */}
      <Box p={4} borderBottomWidth="1px" borderColor={borderColor}>
        <HStack justify="space-between" wrap="wrap" spacing={4}>
          <HStack spacing={4}>
            <InputGroup maxW="300px">
              <InputLeftElement pointerEvents="none">
                <FiSearch />
              </InputLeftElement>
              <Input
                placeholder="Search all columns..."
                value={globalFilter}
                onChange={(e) => setGlobalFilter(e.target.value)}
              />
            </InputGroup>
            
            <Button
              leftIcon={<FiPlus />}
              colorScheme="blue"
              onClick={addNewRow}
              size="sm"
            >
              Add Row
            </Button>
          </HStack>

          <HStack spacing={2}>
            {Object.keys(selectedRows).filter(id => selectedRows[id]).length > 0 && (
              <>
                <Badge colorScheme="blue" px={2} py={1}>
                  {Object.keys(selectedRows).filter(id => selectedRows[id]).length} selected
                </Badge>
                
                <IconButton
                  icon={<FiCopy />}
                  onClick={handleCopy}
                  size="sm"
                  aria-label="Copy selected"
                  variant="ghost"
                />
                
                <IconButton
                  icon={<FiTrash2 />}
                  onClick={onDeleteOpen}
                  size="sm"
                  aria-label="Delete selected"
                  variant="ghost"
                  colorScheme="red"
                />
              </>
            )}
            
            <IconButton
              icon={<FiClipboard />}
              onClick={handlePaste}
              size="sm"
              aria-label="Paste from clipboard"
              variant="ghost"
            />
            
            <Menu>
              <MenuButton
                as={IconButton}
                icon={<FiMoreVertical />}
                size="sm"
                variant="ghost"
              />
              <MenuList>
                <MenuItem icon={<FiDownload />} onClick={exportData}>
                  Export CSV
                </MenuItem>
                <MenuItem icon={<FiUpload />} onClick={() => fileInputRef.current?.click()}>
                  Import CSV
                </MenuItem>
              </MenuList>
            </Menu>
            
            {enableRealtime && (
              <Tooltip label={isConnected ? 'Real-time updates active' : 'Connecting...'}>
                <Badge
                  colorScheme={isConnected ? 'green' : 'yellow'}
                  variant="subtle"
                  px={2}
                  py={1}
                >
                  {isConnected ? 'Live' : 'Connecting'}
                </Badge>
              </Tooltip>
            )}
          </HStack>
        </HStack>
      </Box>

      {/* Table */}
      <Box overflowX="auto" maxH="70vh" overflowY="auto">
        <Table size="sm">
          <Thead position="sticky" top={0} bg={bgColor} zIndex={1}>
            {table.getHeaderGroups().map((headerGroup) => (
              <Tr key={headerGroup.id}>
                <Th borderColor={borderColor}>
                  <Checkbox
                    isChecked={table.getIsAllRowsSelected()}
                    isIndeterminate={table.getIsSomeRowsSelected()}
                    onChange={table.getToggleAllRowsSelectedHandler()}
                  />
                </Th>
                {headerGroup.headers.map((header) => (
                  <Th
                    key={header.id}
                    borderColor={borderColor}
                    cursor="pointer"
                    userSelect="none"
                    onClick={header.column.getToggleSortingHandler()}
                  >
                    <HStack spacing={1}>
                      {flexRender(
                        header.column.columnDef.header,
                        header.getContext()
                      )}
                      {header.column.getIsSorted() && (
                        <Box>
                          {header.column.getIsSorted() === 'desc' ? (
                            <FiChevronDown />
                          ) : (
                            <FiChevronUp />
                          )}
                        </Box>
                      )}
                    </HStack>
                  </Th>
                ))}
              </Tr>
            ))}
          </Thead>
          <Tbody>
            {table.getRowModel().rows.map((row) => (
              <Tr
                key={row.id}
                bg={row.getIsSelected() ? selectedBg : undefined}
                _hover={{ bg: hoverBg }}
              >
                <Td borderColor={borderColor}>
                  <Checkbox
                    isChecked={row.getIsSelected()}
                    onChange={row.getToggleSelectedHandler()}
                  />
                </Td>
                {row.getVisibleCells().map((cell) => (
                  <Td key={cell.id} borderColor={borderColor} p={0}>
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </Td>
                ))}
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>

      {/* Pagination */}
      <Box p={4} borderTopWidth="1px" borderColor={borderColor}>
        <HStack justify="space-between">
          <Text fontSize="sm" color="gray.500">
            Showing {table.getState().pagination.pageIndex * pageSize + 1} to{' '}
            {Math.min(
              (table.getState().pagination.pageIndex + 1) * pageSize,
              data.length
            )}{' '}
            of {data.length} entries
          </Text>
          
          <HStack spacing={2}>
            <Button
              size="sm"
              onClick={() => table.previousPage()}
              isDisabled={!table.getCanPreviousPage()}
            >
              Previous
            </Button>
            <Button
              size="sm"
              onClick={() => table.nextPage()}
              isDisabled={!table.getCanNextPage()}
            >
              Next
            </Button>
          </HStack>
        </HStack>
      </Box>

      {/* Delete confirmation modal */}
      <Modal isOpen={isDeleteOpen} onClose={onDeleteClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Delete Selected Rows</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Are you sure you want to delete{' '}
            {Object.keys(selectedRows).filter(id => selectedRows[id]).length} rows?
            This action cannot be undone.
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" onClick={onDeleteClose} mr={3}>
              Cancel
            </Button>
            <Button colorScheme="red" onClick={deleteSelectedRows}>
              Delete
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Hidden file input for CSV import */}
      <input
        ref={fileInputRef}
        type="file"
        accept=".csv"
        style={{ display: 'none' }}
        onChange={importData}
      />
    </AnimatedCard>
  )
}

export default SpreadsheetEditor