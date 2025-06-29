/**
 * Deployd Client Library
 * Compatible with original deployd API
 * 
 * Usage:
 *   const dpd = new Deployd('http://localhost:2403');
 *   dpd.users.get({}, (err, users) => console.log(users));
 *   dpd.users.on('created', user => console.log('New user:', user));
 */

(function(global) {
    'use strict';

    // WebSocket connection states
    const CONNECTING = 0;
    const OPEN = 1;
    const CLOSING = 2;
    const CLOSED = 3;

    /**
     * Main Deployd client class
     */
    class Deployd {
        constructor(url, options = {}) {
            this.url = url || (typeof window !== 'undefined' ? window.location.origin : 'http://localhost:2403');
            this.options = options;
            this.collections = {};
            this.socket = null;
            this.socketReady = false;
            this.authToken = null;
            this.eventHandlers = new Map();
            this.messageId = 0;
            this.pendingRequests = new Map();
            
            // Auto-connect WebSocket if not disabled
            if (options.realtime !== false) {
                this.connectWebSocket();
            }

            // Initialize collections as proxy properties
            return new Proxy(this, {
                get: (target, prop) => {
                    if (prop in target) {
                        return target[prop];
                    }
                    
                    // Create collection on-demand
                    if (typeof prop === 'string' && !prop.startsWith('_')) {
                        if (!target.collections[prop]) {
                            target.collections[prop] = new Collection(prop, target);
                        }
                        return target.collections[prop];
                    }
                    
                    return target[prop];
                }
            });
        }

        /**
         * Connect to WebSocket for real-time features
         */
        connectWebSocket() {
            if (typeof WebSocket === 'undefined') {
                console.warn('WebSocket not available, real-time features disabled');
                return;
            }

            const wsUrl = this.url.replace(/^http/, 'ws') + '/socket.io/';
            this.socket = new WebSocket(wsUrl);

            this.socket.onopen = () => {
                console.log('WebSocket connected');
                this.socketReady = true;
                
                // Authenticate if we have a token
                if (this.authToken) {
                    this.sendWebSocketMessage({
                        type: 'auth',
                        token: this.authToken
                    });
                }
                
                this.emit('connect');
            };

            this.socket.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleWebSocketMessage(message);
                } catch (err) {
                    console.error('Failed to parse WebSocket message:', err);
                }
            };

            this.socket.onclose = () => {
                console.log('WebSocket disconnected');
                this.socketReady = false;
                this.emit('disconnect');
                
                // Attempt to reconnect after 3 seconds
                setTimeout(() => {
                    if (this.socket.readyState === CLOSED) {
                        this.connectWebSocket();
                    }
                }, 3000);
            };

            this.socket.onerror = (error) => {
                console.error('WebSocket error:', error);
                this.emit('error', error);
            };
        }

        /**
         * Handle incoming WebSocket messages
         */
        handleWebSocketMessage(message) {
            const { type, event, data, room, error } = message;

            if (error) {
                console.error('WebSocket error:', error);
                return;
            }

            switch (type) {
                case 'connect':
                    console.log('WebSocket connection confirmed');
                    break;
                    
                case 'auth':
                    if (data.authenticated) {
                        console.log('WebSocket authentication successful');
                        this.emit('authenticated', data);
                    }
                    break;
                    
                case 'emit':
                    // Handle collection events from emit messages
                    if (room && room.startsWith('collection:')) {
                        this.handleCollectionChange(room, event, data);
                    } else {
                        this.emit(event, data, room);
                    }
                    break;
                    
                case 'collection:change':
                    this.handleCollectionChange(room, event, data);
                    break;
                    
                default:
                    console.log('Unknown WebSocket message type:', type);
            }
        }

        /**
         * Handle collection change events
         */
        handleCollectionChange(collection, eventType, data) {
            // Remove 'collection:' prefix if present
            const collectionName = collection.replace(/^collection:/, '');
            
            // Emit to collection-specific listeners
            this.emit(`${collectionName}:${eventType}`, data);
            this.emit(`${collectionName}:changed`, { type: eventType, data });
            
            // Emit to global collection listeners
            this.emit('collection:changed', { collection: collectionName, type: eventType, data });
        }

        /**
         * Send a WebSocket message
         */
        sendWebSocketMessage(message) {
            if (this.socket && this.socket.readyState === OPEN) {
                this.socket.send(JSON.stringify(message));
            }
        }

        /**
         * Join a WebSocket room for real-time updates
         */
        join(room, callback) {
            if (this.socketReady) {
                this.sendWebSocketMessage({
                    type: 'join',
                    room: room
                });
                if (callback) callback();
            } else {
                this.once('connect', () => this.join(room, callback));
            }
        }

        /**
         * Leave a WebSocket room
         */
        leave(room, callback) {
            if (this.socketReady) {
                this.sendWebSocketMessage({
                    type: 'leave',
                    room: room
                });
                if (callback) callback();
            }
        }

        /**
         * Emit a custom event
         */
        socketEmit(event, data, room) {
            if (this.socketReady) {
                this.sendWebSocketMessage({
                    type: 'emit',
                    event: event,
                    data: data,
                    room: room
                });
            }
        }

        /**
         * Event emitter functionality
         */
        on(event, handler) {
            if (!this.eventHandlers.has(event)) {
                this.eventHandlers.set(event, []);
            }
            this.eventHandlers.get(event).push(handler);
        }

        once(event, handler) {
            const wrapper = (...args) => {
                handler(...args);
                this.off(event, wrapper);
            };
            this.on(event, wrapper);
        }

        off(event, handler) {
            if (this.eventHandlers.has(event)) {
                const handlers = this.eventHandlers.get(event);
                const index = handlers.indexOf(handler);
                if (index > -1) {
                    handlers.splice(index, 1);
                }
            }
        }

        emit(event, ...args) {
            if (this.eventHandlers.has(event)) {
                this.eventHandlers.get(event).forEach(handler => {
                    try {
                        handler(...args);
                    } catch (err) {
                        console.error('Error in event handler:', err);
                    }
                });
            }
        }

        /**
         * Set authentication token
         */
        setAuthToken(token) {
            this.authToken = token;
            if (this.socketReady) {
                this.sendWebSocketMessage({
                    type: 'auth',
                    token: token
                });
            }
        }

        /**
         * Set JWT token (alias for setAuthToken)
         */
        setToken(token) {
            return this.setAuthToken(token);
        }

        /**
         * HTTP request helper
         */
        request(method, path, data, callback) {
            if (typeof data === 'function') {
                callback = data;
                data = null;
            }

            const url = `${this.url}${path}`;
            const options = {
                method: method,
                headers: {
                    'Content-Type': 'application/json'
                }
            };

            // Add authentication header if available
            if (this.authToken) {
                options.headers['Authorization'] = `Bearer ${this.authToken}`;
            }

            if (data) {
                options.body = JSON.stringify(data);
            }

            fetch(url, options)
                .then(response => {
                    if (!response.ok) {
                        return response.text().then(text => {
                            try {
                                const error = JSON.parse(text);
                                throw new Error(error.error || error.message || `HTTP ${response.status}`);
                            } catch {
                                throw new Error(text || `HTTP ${response.status}`);
                            }
                        });
                    }
                    return response.json();
                })
                .then(result => {
                    if (callback) callback(null, result);
                })
                .catch(error => {
                    if (callback) callback(error);
                });
        }
    }

    /**
     * Collection class for interacting with collections
     */
    class Collection {
        constructor(name, client) {
            this.name = name;
            this.client = client;
            this.eventHandlers = new Map();
            
            // Auto-join collection room for real-time updates
            if (client.options.realtime !== false) {
                client.join(`collection:${name}`);
            }
        }

        /**
         * Get documents from collection
         */
        get(query, callback) {
            if (typeof query === 'function') {
                callback = query;
                query = {};
            }
            
            const queryString = Object.keys(query).length > 0 ? 
                '?' + new URLSearchParams(this.flattenQuery(query)).toString() : '';
            
            this.client.request('GET', `/${this.name}${queryString}`, callback);
        }

        /**
         * Create a new document
         */
        post(data, callback) {
            this.client.request('POST', `/${this.name}`, data, callback);
        }

        /**
         * Update a document
         */
        put(id, data, callback) {
            if (typeof id === 'object') {
                // Update by query
                callback = data;
                data = id;
                this.client.request('PUT', `/${this.name}`, data, callback);
            } else {
                // Update by ID
                this.client.request('PUT', `/${this.name}/${id}`, data, callback);
            }
        }

        /**
         * Delete a document
         */
        del(id, callback) {
            if (typeof id === 'object') {
                // Delete by query
                callback = id;
                this.client.request('DELETE', `/${this.name}`, callback);
            } else {
                // Delete by ID
                this.client.request('DELETE', `/${this.name}/${id}`, callback);
            }
        }

        // Alias for del
        delete(id, callback) {
            return this.del(id, callback);
        }

        /**
         * Listen for collection events
         */
        on(event, handler) {
            // Map original deployd events to new format
            const eventMap = {
                'changed': `${this.name}:changed`,
                'created': `${this.name}:created`,
                'updated': `${this.name}:updated`,
                'deleted': `${this.name}:deleted`
            };

            const mappedEvent = eventMap[event] || `${this.name}:${event}`;
            this.client.on(mappedEvent, handler);
        }

        once(event, handler) {
            const eventMap = {
                'changed': `${this.name}:changed`,
                'created': `${this.name}:created`,
                'updated': `${this.name}:updated`,
                'deleted': `${this.name}:deleted`
            };

            const mappedEvent = eventMap[event] || `${this.name}:${event}`;
            this.client.once(mappedEvent, handler);
        }

        off(event, handler) {
            const eventMap = {
                'changed': `${this.name}:changed`,
                'created': `${this.name}:created`,
                'updated': `${this.name}:updated`,
                'deleted': `${this.name}:deleted`
            };

            const mappedEvent = eventMap[event] || `${this.name}:${event}`;
            this.client.off(mappedEvent, handler);
        }

        /**
         * Flatten query object for URL parameters
         */
        flattenQuery(obj, prefix = '') {
            const flattened = {};
            for (const key in obj) {
                if (obj.hasOwnProperty(key)) {
                    const value = obj[key];
                    const newKey = prefix ? `${prefix}.${key}` : key;
                    
                    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                        Object.assign(flattened, this.flattenQuery(value, newKey));
                    } else {
                        flattened[newKey] = JSON.stringify(value);
                    }
                }
            }
            return flattened;
        }
    }

    // Export for different environments
    if (typeof module !== 'undefined' && module.exports) {
        // Node.js
        module.exports = Deployd;
    } else if (typeof define === 'function' && define.amd) {
        // AMD
        define(() => Deployd);
    } else {
        // Browser global
        global.Deployd = Deployd;
        
        // Also create global dpd instance for compatibility
        if (typeof window !== 'undefined') {
            global.dpd = new Deployd();
        }
    }

})(typeof window !== 'undefined' ? window : global);