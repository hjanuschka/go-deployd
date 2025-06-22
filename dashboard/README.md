# go-deployd Dashboard

Modern Material UI dashboard for go-deployd, inspired by the original deployd dashboard but with a sleek, modern design.

## Features

- 🎨 **Material UI 5** - Modern, responsive design
- 📊 **Dashboard Overview** - Server stats and collection summary
- 🗃️ **Collection Management** - Create, edit, and manage collections
- 📋 **Data Browser** - View, edit, add, and delete documents with inline editing
- ⚙️ **Schema Editor** - Visual property editor with validation
- 🔧 **API Tester** - Test endpoints directly from the dashboard
- 📱 **Responsive** - Works on desktop, tablet, and mobile

## Development

```bash
# Install dependencies
npm install

# Start development server (separate from go-deployd)
npm run dev

# Build for production
npm run build
```

## Dashboard Sections

### 📊 Dashboard
- Server overview and statistics
- Collection summary
- System information
- Real-time metrics

### 🗃️ Collections
- Visual collection browser
- Create new collections
- Edit collection schemas
- Delete collections with confirmation

### 📋 Collection Detail
**Data Tab:**
- Browse documents in a data grid
- Add new documents with form validation
- Edit existing documents inline
- Delete documents with confirmation
- Filter and search data

**Properties Tab:**
- Visual schema editor
- Add/edit/remove properties
- Set field types, defaults, and validation
- Required field configuration

**API Tab:**
- Generated API documentation
- Endpoint examples
- Query parameter reference

### 🔧 API Tester
- Interactive API testing tool
- Pre-built example requests
- Custom request builder
- Response viewer with syntax highlighting
- HTTP method selection
- Header and body editors

### ⚙️ Settings
- Server configuration
- Database settings
- Environment information
- API endpoint reference

## Technology Stack

- **React 18** - Modern React with hooks
- **Material UI 5** - Component library and theming
- **Vite** - Fast build tool and dev server
- **Axios** - HTTP client for API calls
- **React Router** - Client-side routing
- **MUI X Data Grid** - Advanced data table
- **Date Fns** - Date handling utilities

## API Integration

The dashboard communicates with go-deployd through:
- REST API for data operations (`/todos`, `/users`, etc.)
- Admin API for management (`/_admin/collections`, `/_admin/info`)
- WebSocket for real-time updates (planned)

## Building

The dashboard is built into the `../web/dashboard/` directory and served by the Go server at `/_dashboard/`.

In development mode, go-deployd automatically redirects `/` to `/_dashboard/` for easy access.