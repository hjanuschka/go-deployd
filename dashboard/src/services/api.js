import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

// Add request interceptor to include JWT token from localStorage if available
api.interceptors.request.use(
  (config) => {
    // Get JWT token from localStorage
    const authToken = localStorage.getItem('authToken')
    if (authToken) {
      config.headers['Authorization'] = `Bearer ${authToken}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Add response interceptor for error handling
api.interceptors.response.use(
  (response) => {
    console.log('API Response:', response.status, response.config.url, response.data)
    return response
  },
  (error) => {
    console.error('API Error:', error)
    console.error('Error details:', error.response?.status, error.response?.data)
    
    // If we get 401, clear stored auth token and user data
    if (error.response?.status === 401) {
      localStorage.removeItem('authToken')
      localStorage.removeItem('authUser')
      // Optionally redirect to login page
      window.location.href = '/_dashboard/login'
    }
    
    return Promise.reject(error)
  }
)

export const apiService = {
  // Generic HTTP methods
  get: (url, config) => api.get(url, config),
  post: (url, data, config) => api.post(url, data, config),
  put: (url, data, config) => api.put(url, data, config),
  delete: (url, config) => api.delete(url, config),

  // Collection operations
  getCollections: async () => {
    const response = await api.get('/_admin/collections')
    return response.data
  },

  getCollection: async (name) => {
    const response = await api.get(`/_admin/collections/${name}`)
    return response.data
  },

  createCollection: async (name, config) => {
    const response = await api.post(`/_admin/collections/${name}`, config)
    return response.data
  },

  updateCollection: async (name, config) => {
    console.log('apiService.updateCollection called')
    console.log('Collection name:', name)
    console.log('Config to send:', config)
    console.log('URL:', `/_admin/collections/${name}`)
    
    const response = await api.put(`/_admin/collections/${name}`, config)
    console.log('API response:', response)
    return response.data
  },

  deleteCollection: async (name) => {
    const response = await api.delete(`/_admin/collections/${name}`)
    return response.data
  },

  // Data operations
  getCollectionData: async (name, params = {}) => {
    // Add $skipEvents for admin dashboard to bypass event validation
    const requestParams = { ...params, $skipEvents: true }
    const response = await api.get(`/${name}`, { params: requestParams })
    return response.data
  },

  createDocument: async (collection, data) => {
    // Add $skipEvents for admin dashboard to bypass event validation
    const requestData = { ...data, $skipEvents: true }
    const response = await api.post(`/${collection}`, requestData)
    return response.data
  },

  updateDocument: async (collection, id, data) => {
    // Add $skipEvents for admin dashboard to bypass event validation
    // Remove id from data since it's in the URL
    const { id: _, ...dataWithoutId } = data
    const requestData = { ...dataWithoutId, $skipEvents: true }
    const response = await api.put(`/${collection}/${id}`, requestData)
    return response.data
  },

  deleteDocument: async (collection, id) => {
    // Add $skipEvents for admin dashboard to bypass event validation
    const response = await api.delete(`/${collection}/${id}?$skipEvents=true`)
    return response.data
  },

  getDocument: async (collection, id) => {
    // Add $skipEvents for admin dashboard to bypass event validation
    const response = await api.get(`/${collection}/${id}?$skipEvents=true`)
    return response.data
  },

  // Count documents
  getDocumentCount: async (collection, query = {}) => {
    // Add $skipEvents for admin dashboard to bypass event validation
    const requestParams = { ...query, $skipEvents: true }
    const response = await api.get(`/${collection}/count`, { params: requestParams })
    return response.data
  },

  // MongoDB-style query execution
  queryCollection: async (collection, mongoQuery, options = {}) => {
    // Add $skipEvents for admin dashboard to bypass event validation
    // Flatten the mongoQuery and options into query parameters
    const requestParams = {
      ...mongoQuery,  // Spread the query conditions directly as parameters
      ...options,     // Add options like $sort, $limit, $skip, $fields
      $skipEvents: true
    }
    
    const response = await api.get(`/${collection}`, { params: requestParams })
    return response.data
  },

  // Server info
  getServerInfo: async () => {
    const response = await api.get('/_admin/info')
    return response.data
  },

  // Test API endpoint
  testEndpoint: async (method, url, data = null, headers = {}) => {
    const config = { headers }
    switch (method.toLowerCase()) {
      case 'get':
        return api.get(url, config)
      case 'post':
        return api.post(url, data, config)
      case 'put':
        return api.put(url, data, config)
      case 'delete':
        return api.delete(url, config)
      default:
        throw new Error(`Unsupported HTTP method: ${method}`)
    }
  },

  // Event script management
  getCollectionEvents: async (collection) => {
    const response = await api.get(`/_admin/collections/${collection}/events`)
    return response.data
  },

  updateCollectionEvent: async (collection, eventName, script, scriptType = 'js') => {
    const response = await api.put(`/_admin/collections/${collection}/events/${eventName}`, {
      script,
      type: scriptType
    })
    return response.data
  },

  updateCollectionConfig: async (collection, config) => {
    const response = await api.put(`/_admin/collections/${collection}/config`, config)
    return response.data
  },

  testCollectionEvent: async (collection, eventName, testContext) => {
    const response = await api.post(`/_admin/collections/${collection}/events/${eventName}/test`, testContext)
    return response.data
  }
}

export default api