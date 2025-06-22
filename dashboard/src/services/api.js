import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

// Add response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error)
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
    const response = await api.put(`/_admin/collections/${name}`, config)
    return response.data
  },

  deleteCollection: async (name) => {
    const response = await api.delete(`/_admin/collections/${name}`)
    return response.data
  },

  // Data operations
  getCollectionData: async (name, params = {}) => {
    const response = await api.get(`/${name}`, { params })
    return response.data
  },

  createDocument: async (collection, data) => {
    const response = await api.post(`/${collection}`, data)
    return response.data
  },

  updateDocument: async (collection, id, data) => {
    const response = await api.put(`/${collection}/${id}`, data)
    return response.data
  },

  deleteDocument: async (collection, id) => {
    const response = await api.delete(`/${collection}/${id}`)
    return response.data
  },

  getDocument: async (collection, id) => {
    const response = await api.get(`/${collection}/${id}`)
    return response.data
  },

  // Count documents
  getDocumentCount: async (collection, query = {}) => {
    const response = await api.get(`/${collection}/count`, { params: query })
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