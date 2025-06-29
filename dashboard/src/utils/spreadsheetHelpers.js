// Utility functions for the SpreadsheetEditor component

export const parseCSV = (text, columns) => {
  const lines = text.split('\n').filter(line => line.trim())
  const headers = lines[0].split(',').map(h => h.trim())
  
  // Map CSV headers to column accessorKeys
  const headerMap = {}
  headers.forEach((header, index) => {
    const column = columns.find(col => 
      col.header.toLowerCase() === header.toLowerCase() ||
      col.accessorKey.toLowerCase() === header.toLowerCase()
    )
    if (column) {
      headerMap[index] = column.accessorKey
    }
  })
  
  // Parse data rows
  const data = []
  for (let i = 1; i < lines.length; i++) {
    const values = lines[i].split(',').map(v => v.trim())
    const row = {}
    
    values.forEach((value, index) => {
      const key = headerMap[index]
      if (key) {
        const column = columns.find(col => col.accessorKey === key)
        const dataType = column?.meta?.dataType || 'string'
        
        // Convert value based on data type
        switch (dataType) {
          case 'number':
            row[key] = value ? parseFloat(value) : 0
            break
          case 'boolean':
            row[key] = value.toLowerCase() === 'true' || value === '1'
            break
          case 'date':
            row[key] = value || null
            break
          default:
            row[key] = value
        }
      }
    })
    
    if (Object.keys(row).length > 0) {
      data.push(row)
    }
  }
  
  return data
}

export const exportToCSV = (data, columns, filename) => {
  // Create CSV header
  const headers = columns.map(col => col.header).join(',')
  
  // Create CSV rows
  const rows = data.map(row => {
    return columns.map(col => {
      const value = row[col.accessorKey]
      
      // Handle different data types
      if (value === null || value === undefined) return ''
      if (typeof value === 'boolean') return value ? 'true' : 'false'
      if (typeof value === 'string' && value.includes(',')) {
        return `"${value.replace(/"/g, '""')}"` // Escape quotes and wrap in quotes
      }
      return value
    }).join(',')
  })
  
  // Combine header and rows
  const csv = [headers, ...rows].join('\n')
  
  // Download file
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  
  link.setAttribute('href', url)
  link.setAttribute('download', filename)
  link.style.visibility = 'hidden'
  
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  
  URL.revokeObjectURL(url)
}

export const getNextCell = (currentRowIndex, currentColumnId, direction, table) => {
  const visibleColumns = table.getVisibleFlatColumns()
  const rows = table.getRowModel().rows
  
  const currentColumnIndex = visibleColumns.findIndex(col => col.id === currentColumnId)
  
  switch (direction) {
    case 'ArrowRight':
    case 'Tab':
      if (currentColumnIndex < visibleColumns.length - 1) {
        return {
          rowIndex: currentRowIndex,
          columnId: visibleColumns[currentColumnIndex + 1].id
        }
      } else if (currentRowIndex < rows.length - 1) {
        return {
          rowIndex: currentRowIndex + 1,
          columnId: visibleColumns[0].id
        }
      }
      break
      
    case 'ArrowLeft':
    case 'ShiftTab':
      if (currentColumnIndex > 0) {
        return {
          rowIndex: currentRowIndex,
          columnId: visibleColumns[currentColumnIndex - 1].id
        }
      } else if (currentRowIndex > 0) {
        return {
          rowIndex: currentRowIndex - 1,
          columnId: visibleColumns[visibleColumns.length - 1].id
        }
      }
      break
      
    case 'ArrowDown':
    case 'Enter':
      if (currentRowIndex < rows.length - 1) {
        return {
          rowIndex: currentRowIndex + 1,
          columnId: currentColumnId
        }
      }
      break
      
    case 'ArrowUp':
      if (currentRowIndex > 0) {
        return {
          rowIndex: currentRowIndex - 1,
          columnId: currentColumnId
        }
      }
      break
  }
  
  return null
}

export const validateData = (data, columns) => {
  const errors = []
  
  data.forEach((row, index) => {
    columns.forEach(column => {
      const value = row[column.accessorKey]
      const dataType = column.meta?.dataType
      const required = column.meta?.required
      
      // Check required fields
      if (required && (value === null || value === undefined || value === '')) {
        errors.push({
          row: index,
          column: column.accessorKey,
          message: `${column.header} is required`
        })
      }
      
      // Validate data types
      if (value !== null && value !== undefined && value !== '') {
        switch (dataType) {
          case 'number':
            if (isNaN(value)) {
              errors.push({
                row: index,
                column: column.accessorKey,
                message: `${column.header} must be a number`
              })
            }
            break
            
          case 'email':
            const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
            if (!emailRegex.test(value)) {
              errors.push({
                row: index,
                column: column.accessorKey,
                message: `${column.header} must be a valid email`
              })
            }
            break
            
          case 'date':
            if (isNaN(Date.parse(value))) {
              errors.push({
                row: index,
                column: column.accessorKey,
                message: `${column.header} must be a valid date`
              })
            }
            break
        }
      }
    })
  })
  
  return errors
}

export const formatCellValue = (value, dataType) => {
  if (value === null || value === undefined) return ''
  
  switch (dataType) {
    case 'number':
      return typeof value === 'number' ? value.toLocaleString() : value
      
    case 'currency':
      return typeof value === 'number' 
        ? new Intl.NumberFormat('en-US', {
            style: 'currency',
            currency: 'USD'
          }).format(value)
        : value
        
    case 'percent':
      return typeof value === 'number'
        ? `${(value * 100).toFixed(2)}%`
        : value
        
    case 'date':
      try {
        return new Date(value).toLocaleDateString()
      } catch {
        return value
      }
      
    case 'datetime':
      try {
        return new Date(value).toLocaleString()
      } catch {
        return value
      }
      
    default:
      return value
  }
}