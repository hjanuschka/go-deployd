# dpd.js Client Library

dpd.js is the official JavaScript client library for go-deployd. It provides a simple, jQuery-like API for working with collections, real-time events, and authentication.

## Table of Contents

- [Installation & Setup](#installation--setup)
- [Authentication](#authentication)
- [Collection Operations](#collection-operations)
  - [Basic CRUD Operations](#basic-crud-operations)
  - [Advanced Querying](#advanced-querying)
  - [Real-time Collection Updates](#real-time-collection-updates)
- [Promise-based API](#promise-based-api)
- [Error Handling](#error-handling)
- [Custom Events](#custom-events)
- [Complete Example App](#complete-example-app)

## Installation & Setup

### Install via npm

```bash
# Install dpd.js
npm install dpd

# Or include via CDN
<script src="https://unpkg.com/dpd/dpd.js"></script>
```

### Basic Setup

```javascript
// Initialize dpd client
const dpd = require('dpd');

// Set the server URL (defaults to current domain)
dpd.setBaseURL('http://localhost:2403');

// Access collections
const todos = dpd.todos;
const users = dpd.users;
```

### Browser Setup

```html
<!DOCTYPE html>
<html>
<head>
    <title>Go-Deployd App</title>
    <script src="https://unpkg.com/dpd/dpd.js"></script>
</head>
<body>
    <script>
        // dpd is available globally
        console.log('Connected to:', dpd.baseURL);
        
        // Use collections
        dpd.todos.get(function(todos) {
            console.log('Todos:', todos);
        });
    </script>
</body>
</html>
```

## Authentication

```javascript
// Login with username/password
dpd.users.login({
    username: 'user@example.com',
    password: 'password123'
}, function(result, error) {
    if (error) {
        console.error('Login failed:', error);
    } else {
        console.log('Logged in:', result);
        // User is now authenticated for subsequent requests
    }
});

// Login with master key
dpd.users.login({
    masterKey: 'mk_your_master_key_here'
}, function(result, error) {
    if (error) {
        console.error('Master key login failed:', error);
    } else {
        console.log('Logged in as root:', result);
    }
});

// Check current user
dpd.users.me(function(user) {
    if (user) {
        console.log('Current user:', user);
    } else {
        console.log('Not logged in');
    }
});

// Logout
dpd.users.logout();
```

## Collection Operations

### Basic CRUD Operations

```javascript
// CREATE - Add new document
dpd.todos.post({
    title: 'Learn go-deployd',
    completed: false,
    priority: 5
}, function(result, error) {
    if (error) {
        console.error('Create failed:', error);
    } else {
        console.log('Created todo:', result);
    }
});

// READ - Get all documents
dpd.todos.get(function(todos, error) {
    if (error) {
        console.error('Get failed:', error);
    } else {
        console.log('All todos:', todos);
    }
});

// READ - Get single document by ID
dpd.todos.get('doc123', function(todo, error) {
    if (error) {
        console.error('Get failed:', error);
    } else {
        console.log('Todo:', todo);
    }
});

// UPDATE - Update document
dpd.todos.put('doc123', {
    title: 'Updated title',
    completed: true
}, function(result, error) {
    if (error) {
        console.error('Update failed:', error);
    } else {
        console.log('Updated todo:', result);
    }
});

// DELETE - Remove document
dpd.todos.del('doc123', function(result, error) {
    if (error) {
        console.error('Delete failed:', error);
    } else {
        console.log('Deleted todo');
    }
});
```

### Advanced Querying

```javascript
// Simple filtering
dpd.todos.get({
    completed: false,
    priority: {$gte: 5}
}, function(todos) {
    console.log('High priority incomplete todos:', todos);
});

// Complex queries with $or
dpd.todos.get({
    $or: [
        {priority: {$gte: 8}},
        {title: {$regex: 'urgent'}}
    ]
}, function(todos) {
    console.log('Urgent todos:', todos);
});

// Sorting and pagination
dpd.todos.get({
    $sort: {createdAt: -1}, // Sort by created date, newest first
    $limit: 10,             // Limit to 10 results
    $skip: 0                // Skip 0 (for pagination)
}, function(todos) {
    console.log('Recent todos:', todos);
});

// Field projection
dpd.todos.get({
    completed: false,
    $fields: {title: 1, priority: 1} // Only return title and priority
}, function(todos) {
    console.log('Minimal todo data:', todos);
});
```

### Real-time Collection Updates

```javascript
// Listen for real-time updates on todos collection
dpd.todos.on('created', function(todo) {
    console.log('New todo created:', todo);
    addTodoToUI(todo);
});

dpd.todos.on('updated', function(todo) {
    console.log('Todo updated:', todo);
    updateTodoInUI(todo);
});

dpd.todos.on('deleted', function(todo) {
    console.log('Todo deleted:', todo);
    removeTodoFromUI(todo.id);
});

// Listen for all changes
dpd.todos.on('changed', function(todo, action) {
    console.log(`Todo ${action}:`, todo);
    switch(action) {
        case 'created':
            addTodoToUI(todo);
            break;
        case 'updated':
            updateTodoInUI(todo);
            break;
        case 'deleted':
            removeTodoFromUI(todo.id);
            break;
    }
});

// Stop listening
dpd.todos.off('created');
dpd.todos.off(); // Remove all listeners
```

## Promise-based API

dpd.js supports both callbacks and promises:

### Using Promises

```javascript
dpd.todos.get({completed: false})
    .then(todos => {
        console.log('Incomplete todos:', todos);
        return dpd.todos.post({
            title: 'New todo from promise',
            completed: false
        });
    })
    .then(newTodo => {
        console.log('Created:', newTodo);
    })
    .catch(error => {
        console.error('Error:', error);
    });
```

### Using async/await

```javascript
async function manageTodos() {
    try {
        // Get all incomplete todos
        const incompleteTodos = await dpd.todos.get({completed: false});
        console.log('Found', incompleteTodos.length, 'incomplete todos');
        
        // Create a new todo
        const newTodo = await dpd.todos.post({
            title: 'New todo from async/await',
            completed: false,
            priority: 3
        });
        
        console.log('Created todo:', newTodo);
        
        // Update it
        const updatedTodo = await dpd.todos.put(newTodo.id, {
            completed: true
        });
        
        console.log('Completed todo:', updatedTodo);
        
    } catch (error) {
        console.error('Error managing todos:', error);
    }
}
```

## Error Handling

### Handle validation errors

```javascript
dpd.todos.post({
    title: '', // Invalid - too short
    priority: 'high' // Invalid - should be number
}, function(result, error) {
    if (error) {
        if (error.statusCode === 400) {
            console.log('Validation errors:');
            error.errors.forEach(err => {
                console.log(`${err.field}: ${err.message}`);
            });
        } else {
            console.error('Unexpected error:', error);
        }
    }
});
```

### Handle authentication errors

```javascript
dpd.todos.get(function(todos, error) {
    if (error) {
        if (error.statusCode === 401) {
            console.log('Authentication required');
            // Redirect to login
        } else if (error.statusCode === 403) {
            console.log('Access forbidden');
        }
    }
});
```

### Global error handling

```javascript
dpd.on('error', function(error) {
    console.error('Global dpd error:', error);
    if (error.statusCode === 401) {
        // Redirect to login page
        window.location.href = '/login';
    }
});
```

## Custom Events

```javascript
// Listen for custom events emitted from server events
dpd.on('urgent_task_created', function(data) {
    console.log('Urgent task alert:', data);
    showNotification('Urgent Task!', data.title);
});

dpd.on('task_completed', function(data) {
    console.log('Task completed:', data);
    showCelebration(data.title);
});

// Send custom events to server (if custom endpoint exists)
dpd.notifications.post({
    type: 'user_action',
    action: 'page_view',
    page: window.location.pathname
});
```

## Complete Example App

```javascript
// Simple todo app using dpd.js
class TodoApp {
    constructor() {
        this.todoList = document.getElementById('todo-list');
        this.todoForm = document.getElementById('todo-form');
        this.setupEventListeners();
        this.loadTodos();
    }
    
    setupEventListeners() {
        // Form submission
        this.todoForm.addEventListener('submit', (e) => {
            e.preventDefault();
            this.addTodo();
        });
        
        // Real-time updates
        dpd.todos.on('created', (todo) => {
            this.addTodoToDOM(todo);
        });
        
        dpd.todos.on('updated', (todo) => {
            this.updateTodoInDOM(todo);
        });
        
        dpd.todos.on('deleted', (todo) => {
            this.removeTodoFromDOM(todo.id);
        });
    }
    
    async loadTodos() {
        try {
            const todos = await dpd.todos.get({
                $sort: {createdAt: -1}
            });
            todos.forEach(todo => this.addTodoToDOM(todo));
        } catch (error) {
            console.error('Failed to load todos:', error);
        }
    }
    
    async addTodo() {
        const titleInput = document.getElementById('todo-title');
        const title = titleInput.value.trim();
        
        if (!title) return;
        
        try {
            await dpd.todos.post({
                title: title,
                completed: false,
                priority: 1
            });
            titleInput.value = '';
        } catch (error) {
            console.error('Failed to create todo:', error);
        }
    }
    
    async toggleTodo(id, completed) {
        try {
            await dpd.todos.put(id, {
                completed: !completed
            });
        } catch (error) {
            console.error('Failed to update todo:', error);
        }
    }
    
    async deleteTodo(id) {
        try {
            await dpd.todos.del(id);
        } catch (error) {
            console.error('Failed to delete todo:', error);
        }
    }
    
    addTodoToDOM(todo) {
        const li = document.createElement('li');
        li.id = `todo-${todo.id}`;
        li.innerHTML = `
            <input type="checkbox" ${todo.completed ? 'checked' : ''} 
                   onchange="app.toggleTodo('${todo.id}', ${todo.completed})">
            <span class="${todo.completed ? 'completed' : ''}">${todo.title}</span>
            <button onclick="app.deleteTodo('${todo.id}')">Delete</button>
        `;
        this.todoList.appendChild(li);
    }
    
    updateTodoInDOM(todo) {
        const li = document.getElementById(`todo-${todo.id}`);
        if (li) {
            li.querySelector('input').checked = todo.completed;
            li.querySelector('span').className = todo.completed ? 'completed' : '';
        }
    }
    
    removeTodoFromDOM(id) {
        const li = document.getElementById(`todo-${id}`);
        if (li) {
            li.remove();
        }
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.app = new TodoApp();
});
```