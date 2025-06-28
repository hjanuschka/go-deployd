# SpreadsheetEditor Component

A production-ready React component for spreadsheet-like data editing with real-time updates, built using @tanstack/react-table and Chakra UI.

## Features

- **Inline Editing**: Click any cell to edit directly
- **Keyboard Navigation**: Full keyboard support with Tab, Enter, and Arrow keys
- **Data Types**: Support for string, number, boolean, date, and select inputs
- **Copy/Paste**: Full clipboard integration for copying and pasting data
- **Sorting & Filtering**: Click column headers to sort, global search across all columns
- **Bulk Actions**: Select multiple rows and perform bulk delete operations
- **Import/Export**: CSV import and export functionality
- **Real-time Updates**: WebSocket integration for live data synchronization
- **Responsive Design**: Works on desktop and mobile devices
- **Error Handling**: Comprehensive error handling with user-friendly messages
- **Loading States**: Smooth loading indicators and skeleton screens

## Usage

```jsx
import SpreadsheetEditor from './components/SpreadsheetEditor'

const MyComponent = () => {
  // Define columns with data types
  const columns = [
    {
      accessorKey: 'id',
      header: 'ID',
      meta: { dataType: 'string' },
    },
    {
      accessorKey: 'name',
      header: 'Name',
      meta: { dataType: 'string', required: true },
    },
    {
      accessorKey: 'email',
      header: 'Email',
      meta: { dataType: 'email', required: true },
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

  return (
    <SpreadsheetEditor
      collection="users"           // Collection name in the database
      columns={columns}            // Column definitions
      onDataChange={handleChange}  // Callback when data changes
      enableRealtime={true}        // Enable WebSocket updates
      pageSize={50}               // Rows per page
    />
  )
}
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `collection` | string | required | The collection/table name to edit |
| `columns` | array | auto-generated | Column definitions with data types |
| `onDataChange` | function | - | Callback fired when data is modified |
| `enableRealtime` | boolean | true | Enable WebSocket real-time updates |
| `pageSize` | number | 50 | Number of rows per page |

## Column Configuration

Each column can have the following properties:

```jsx
{
  accessorKey: 'fieldName',     // Field name in the data
  header: 'Display Name',       // Column header text
  meta: {
    dataType: 'string',         // Data type (see below)
    required: boolean,          // Is field required?
    options: array,             // Options for select type
  }
}
```

### Supported Data Types

- `string`: Text input
- `number`: Numeric input
- `boolean`: Checkbox
- `date`: Date picker
- `email`: Email validation
- `select`: Dropdown selection
- `currency`: Currency formatting
- `percent`: Percentage formatting

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Tab` | Move to next cell |
| `Shift+Tab` | Move to previous cell |
| `Enter` | Edit cell / Move down |
| `Escape` | Cancel editing |
| `Arrow Keys` | Navigate cells |
| `Ctrl/Cmd+C` | Copy selected rows |
| `Ctrl/Cmd+V` | Paste from clipboard |
| `Ctrl/Cmd+S` | Save reminder |
| `Delete` | Delete selected rows |

## API Integration

The component uses the `apiService` for all backend operations:

- `GET /{collection}` - Load data
- `POST /{collection}` - Create new row
- `PUT /{collection}/{id}` - Update row
- `DELETE /{collection}/{id}` - Delete row

All requests include `$skipEvents: true` to bypass event validation in admin mode.

## WebSocket Integration

Real-time updates are handled via WebSocket connection to:
```
ws://host/_ws/collections/{collection}
```

Message format:
```json
{
  "type": "create|update|delete",
  "data": { /* row data */ },
  "id": "row-id"
}
```

## CSV Format

### Export
- Headers use column display names
- Values are properly escaped
- Dates are in ISO format
- Booleans are "true"/"false"

### Import
- First row must contain headers
- Headers match column names (case-insensitive)
- Data types are automatically converted
- Validation is performed before import

## Styling

The component uses Chakra UI theme tokens and respects light/dark mode:

```jsx
// Custom styling
const customTheme = {
  components: {
    Table: {
      variants: {
        spreadsheet: {
          th: { fontSize: 'sm' },
          td: { py: 0 },
        }
      }
    }
  }
}
```

## Error Handling

All errors are displayed via the toast system:
- Network errors
- Validation errors
- Import/export errors
- WebSocket connection errors

## Performance Considerations

- Virtual scrolling for large datasets
- Debounced search input
- Optimistic updates for better UX
- Memoized column definitions
- Batch operations for bulk actions

## Browser Support

- Chrome/Edge: Full support
- Firefox: Full support
- Safari: Full support (clipboard may require permissions)
- Mobile: Touch-optimized with responsive design

## Dependencies

- `@tanstack/react-table`: Table logic and state management
- `@chakra-ui/react`: UI components and styling
- `date-fns`: Date formatting
- `framer-motion`: Animations
- `react-icons`: Icon set

## Future Enhancements

- Undo/redo functionality
- Column resizing and reordering
- Advanced filtering UI
- Formula support
- Collaborative editing indicators
- Excel import/export
- Print view
- Mobile-specific interface