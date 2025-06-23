import React, { useState, useEffect } from 'react'
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormControlLabel,
  Checkbox,
  Box,
  Typography,
  Alert
} from '@mui/material'
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns'

function DocumentEditor({ open, onClose, document, collection, onSave }) {
  const [formData, setFormData] = useState({})
  const [errors, setErrors] = useState({})
  const [skipEvents, setSkipEvents] = useState(false)

  useEffect(() => {
    if (open) {
      if (document) {
        // Editing existing document
        setFormData({ ...document })
      } else {
        // Creating new document
        const initialData = {}
        Object.entries(collection.properties || {}).forEach(([name, property]) => {
          if (property.default !== undefined) {
            if (property.default === 'now' && property.type === 'date') {
              initialData[name] = new Date()
            } else {
              initialData[name] = property.default
            }
          }
        })
        setFormData(initialData)
      }
      setErrors({})
      setSkipEvents(false) // Reset skip events checkbox
    }
  }, [open, document, collection])

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
    
    // Convert date objects to ISO strings
    Object.entries(collection.properties || {}).forEach(([name, property]) => {
      if (property.type === 'date' && processedData[name] instanceof Date) {
        processedData[name] = processedData[name].toISOString()
      }
    })

    // Add $skipEvents flag if checkbox is checked
    if (skipEvents) {
      processedData.$skipEvents = true
    }

    onSave(processedData)
  }

  const renderField = (name, property) => {
    const value = formData[name] || ''
    const error = errors[name]

    switch (property.type) {
      case 'string':
        return (
          <TextField
            key={name}
            label={name}
            value={value}
            onChange={(e) => handleFieldChange(name, e.target.value)}
            fullWidth
            error={!!error}
            helperText={error}
            required={property.required}
          />
        )

      case 'number':
        return (
          <TextField
            key={name}
            label={name}
            type="number"
            value={value}
            onChange={(e) => handleFieldChange(name, parseFloat(e.target.value) || 0)}
            fullWidth
            error={!!error}
            helperText={error}
            required={property.required}
          />
        )

      case 'boolean':
        return (
          <FormControlLabel
            key={name}
            control={
              <Checkbox
                checked={!!value}
                onChange={(e) => handleFieldChange(name, e.target.checked)}
              />
            }
            label={name}
          />
        )

      case 'date':
        return (
          <LocalizationProvider key={name} dateAdapter={AdapterDateFns}>
            <DateTimePicker
              label={name}
              value={value ? new Date(value) : null}
              onChange={(newValue) => handleFieldChange(name, newValue)}
              renderInput={(params) => (
                <TextField
                  {...params}
                  fullWidth
                  error={!!error}
                  helperText={error}
                  required={property.required}
                />
              )}
            />
          </LocalizationProvider>
        )

      case 'array':
        return (
          <TextField
            key={name}
            label={name}
            value={Array.isArray(value) ? value.join(', ') : value}
            onChange={(e) => handleFieldChange(name, e.target.value.split(',').map(v => v.trim()))}
            fullWidth
            error={!!error}
            helperText={error || 'Comma-separated values'}
            required={property.required}
          />
        )

      default:
        return (
          <TextField
            key={name}
            label={name}
            value={value}
            onChange={(e) => handleFieldChange(name, e.target.value)}
            fullWidth
            multiline
            rows={3}
            error={!!error}
            helperText={error}
            required={property.required}
          />
        )
    }
  }

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>
        {document ? 'Edit Document' : 'Create New Document'}
      </DialogTitle>
      <DialogContent>
        <Box display="flex" flexDirection="column" gap={2} sx={{ mt: 2 }}>
          {document && (
            <Alert severity="info">
              Document ID: {document.id}
            </Alert>
          )}
          
          {Object.entries(collection.properties || {}).map(([name, property]) => (
            <Box key={name}>
              <Typography variant="caption" color="textSecondary" gutterBottom>
                {property.type} {property.required && '(required)'}
              </Typography>
              {renderField(name, property)}
            </Box>
          ))}

          {Object.keys(collection.properties || {}).length === 0 && (
            <Alert severity="warning">
              No properties defined for this collection. Add properties in the Properties tab first.
            </Alert>
          )}
          
          <Box mt={2} sx={{ borderTop: '1px solid #e0e0e0', pt: 2 }}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={skipEvents}
                  onChange={(e) => setSkipEvents(e.target.checked)}
                  color="warning"
                />
              }
              label={
                <Box>
                  <Typography variant="body2" fontWeight="bold">
                    Skip Events (Admin Only)
                  </Typography>
                  <Typography variant="caption" color="textSecondary">
                    Bypass all validation and event scripts during save operation. 
                    Requires master key authentication.
                  </Typography>
                </Box>
              }
            />
          </Box>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button 
          onClick={handleSave}
          variant="contained"
          disabled={Object.keys(collection.properties || {}).length === 0}
        >
          {document ? 'Update' : 'Create'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default DocumentEditor