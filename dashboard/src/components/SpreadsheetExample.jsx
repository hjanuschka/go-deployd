import React from 'react'
import { Box, Heading, Text, VStack } from '@chakra-ui/react'
import SpreadsheetEditor from './SpreadsheetEditor'

// Example component showing how to use the SpreadsheetEditor
export const SpreadsheetExample = () => {
  // Define columns with different data types
  const columns = [
    {
      accessorKey: 'id',
      header: 'ID',
      meta: { dataType: 'string' },
      enableSorting: false,
    },
    {
      accessorKey: 'name',
      header: 'Name',
      meta: { dataType: 'string' },
    },
    {
      accessorKey: 'email',
      header: 'Email',
      meta: { dataType: 'string' },
    },
    {
      accessorKey: 'age',
      header: 'Age',
      meta: { dataType: 'number' },
    },
    {
      accessorKey: 'active',
      header: 'Active',
      meta: { dataType: 'boolean' },
    },
    {
      accessorKey: 'role',
      header: 'Role',
      meta: {
        dataType: 'select',
        options: [
          { value: 'admin', label: 'Admin' },
          { value: 'user', label: 'User' },
          { value: 'guest', label: 'Guest' },
        ],
      },
    },
    {
      accessorKey: 'joinedAt',
      header: 'Joined Date',
      meta: { dataType: 'date' },
    },
  ]

  const handleDataChange = () => {
    console.log('Data changed in spreadsheet')
  }

  return (
    <VStack spacing={6} align="stretch">
      <Box>
        <Heading size="lg" mb={2}>
          Spreadsheet Editor Example
        </Heading>
        <Text color="gray.600">
          This spreadsheet editor supports inline editing, keyboard navigation,
          copy/paste, sorting, filtering, and real-time updates.
        </Text>
      </Box>

      <SpreadsheetEditor
        collection="users" // Replace with your collection name
        columns={columns}
        onDataChange={handleDataChange}
        enableRealtime={true}
        pageSize={25}
      />

      <Box>
        <Heading size="md" mb={2}>
          Features:
        </Heading>
        <VStack align="start" spacing={1}>
          <Text>• Click any cell to edit inline</Text>
          <Text>• Use Tab/Enter to navigate between cells</Text>
          <Text>• Select multiple rows with checkboxes</Text>
          <Text>• Copy/paste data with clipboard buttons</Text>
          <Text>• Sort columns by clicking headers</Text>
          <Text>• Search across all columns</Text>
          <Text>• Export data to CSV</Text>
          <Text>• Real-time updates via WebSocket</Text>
        </VStack>
      </Box>
    </VStack>
  )
}

export default SpreadsheetExample