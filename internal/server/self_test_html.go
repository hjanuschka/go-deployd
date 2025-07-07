package server

// SelfTestHTMLContent contains the embedded self-test.html page
const SelfTestHTMLContent = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>go-deployd API Self-Test</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #0f0f0f;
            color: #e0e0e0;
            line-height: 1.6;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
        }

        header {
            background: #1a1a1a;
            padding: 20px 0;
            border-bottom: 1px solid #333;
            margin-bottom: 30px;
        }

        h1 {
            color: #4CAF50;
            text-align: center;
            font-size: 2em;
        }

        .auth-section {
            background: #1a1a1a;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
            border: 1px solid #333;
        }

        .auth-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            margin-bottom: 20px;
        }

        .auth-method {
            background: #242424;
            padding: 20px;
            border-radius: 6px;
            border: 1px solid #333;
        }

        .auth-method h3 {
            color: #4CAF50;
            margin-bottom: 15px;
        }

        .form-group {
            margin-bottom: 15px;
        }

        label {
            display: block;
            margin-bottom: 5px;
            color: #b0b0b0;
            font-size: 0.9em;
        }

        input, select, textarea {
            width: 100%;
            padding: 10px;
            background: #2a2a2a;
            border: 1px solid #444;
            border-radius: 4px;
            color: #e0e0e0;
            font-family: inherit;
        }

        input:focus, select:focus, textarea:focus {
            outline: none;
            border-color: #4CAF50;
        }

        button {
            background: #4CAF50;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
            font-weight: 500;
            transition: all 0.3s;
        }

        button:hover {
            background: #45a049;
            transform: translateY(-1px);
            box-shadow: 0 2px 4px rgba(0,0,0,0.3);
        }

        button:disabled {
            background: #555;
            cursor: not-allowed;
            transform: none;
        }

        .control-panel {
            background: #1a1a1a;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
            border: 1px solid #333;
        }

        .controls {
            display: flex;
            gap: 15px;
            align-items: center;
            flex-wrap: wrap;
        }

        .controls select {
            width: 200px;
        }

        .custom-endpoint {
            display: flex;
            gap: 10px;
            align-items: center;
            flex: 1;
            min-width: 300px;
        }

        .custom-endpoint input {
            flex: 1;
        }

        .test-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .test-card {
            background: #1a1a1a;
            border: 1px solid #333;
            border-radius: 8px;
            padding: 20px;
            transition: all 0.3s;
        }

        .test-card:hover {
            border-color: #4CAF50;
            box-shadow: 0 0 20px rgba(76, 175, 80, 0.1);
        }

        .test-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 15px;
        }

        .test-title {
            font-weight: 600;
            color: #4CAF50;
        }

        .status-badge {
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 0.85em;
            font-weight: 500;
        }

        .status-badge.pending {
            background: #333;
            color: #999;
        }

        .status-badge.success {
            background: #4CAF50;
            color: white;
        }

        .status-badge.error {
            background: #f44336;
            color: white;
        }

        .test-content {
            margin-bottom: 15px;
        }

        .test-actions {
            display: flex;
            gap: 10px;
        }

        .test-actions button {
            font-size: 0.9em;
            padding: 6px 12px;
        }

        .section-title {
            color: #999;
            font-size: 0.85em;
            margin-bottom: 5px;
            text-transform: uppercase;
        }

        .code-block {
            background: #0a0a0a;
            padding: 15px;
            border-radius: 4px;
            font-family: 'Consolas', 'Monaco', monospace;
            font-size: 0.9em;
            overflow-x: auto;
            border: 1px solid #222;
            white-space: pre-wrap;
            word-wrap: break-word;
        }

        .progress-bar {
            height: 4px;
            background: #333;
            border-radius: 2px;
            margin-top: 15px;
            overflow: hidden;
        }

        .progress-fill {
            height: 100%;
            background: #4CAF50;
            transition: width 0.3s;
        }

        .console {
            background: #0a0a0a;
            border: 1px solid #333;
            border-radius: 8px;
            padding: 20px;
            height: 400px;
            overflow-y: auto;
            font-family: 'Consolas', 'Monaco', monospace;
            font-size: 0.9em;
        }

        .log-entry {
            margin-bottom: 8px;
            padding: 5px;
            border-radius: 4px;
        }

        .log-entry.info {
            color: #4CAF50;
        }

        .log-entry.error {
            color: #f44336;
            background: rgba(244, 67, 54, 0.1);
        }

        .log-entry.warning {
            color: #ff9800;
        }

        .timestamp {
            color: #666;
            margin-right: 10px;
        }

        .modal {
            display: none;
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.8);
            z-index: 1000;
            overflow-y: auto;
        }

        .modal-content {
            background: #1a1a1a;
            margin: 50px auto;
            padding: 30px;
            width: 90%;
            max-width: 800px;
            border-radius: 8px;
            border: 1px solid #333;
        }

        .modal-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }

        .modal-header h2 {
            color: #4CAF50;
        }

        .close-button {
            background: none;
            border: none;
            color: #999;
            font-size: 1.5em;
            cursor: pointer;
            padding: 0;
            width: 30px;
            height: 30px;
            display: flex;
            align-items: center;
            justify-content: center;
        }

        .close-button:hover {
            color: #fff;
            transform: none;
        }

        .request-section, .response-section {
            margin-bottom: 20px;
        }

        .json-viewer {
            background: #0a0a0a;
            padding: 15px;
            border-radius: 4px;
            overflow-x: auto;
            border: 1px solid #222;
        }

        .post-data-section {
            background: #1a1a1a;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
            border: 1px solid #333;
        }

        #postData {
            width: 100%;
            min-height: 100px;
            font-family: 'Consolas', 'Monaco', monospace;
            resize: vertical;
        }

        .file-section {
            background: #1a1a1a;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 30px;
            border: 1px solid #333;
        }

        .file-upload-area {
            border: 2px dashed #444;
            border-radius: 8px;
            padding: 40px;
            text-align: center;
            margin-bottom: 20px;
            transition: all 0.3s;
        }

        .file-upload-area.dragover {
            border-color: #4CAF50;
            background: rgba(76, 175, 80, 0.1);
        }

        .file-list {
            margin-top: 20px;
        }

        .file-item {
            background: #242424;
            padding: 15px;
            border-radius: 6px;
            margin-bottom: 10px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border: 1px solid #333;
        }

        .file-info {
            flex: 1;
        }

        .file-name {
            font-weight: 500;
            color: #4CAF50;
            margin-bottom: 5px;
        }

        .file-details {
            font-size: 0.85em;
            color: #999;
        }

        .file-actions {
            display: flex;
            gap: 10px;
        }

        .file-actions button {
            font-size: 0.85em;
            padding: 5px 10px;
        }

        .token-display {
            background: #0a0a0a;
            padding: 10px 15px;
            border-radius: 4px;
            font-family: monospace;
            font-size: 0.85em;
            word-break: break-all;
            margin-top: 10px;
            border: 1px solid #333;
        }

        .auth-status {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 4px;
            font-size: 0.85em;
            font-weight: 500;
            margin-left: 10px;
        }

        .auth-status.authenticated {
            background: #4CAF50;
            color: white;
        }

        .auth-status.unauthenticated {
            background: #666;
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🚀 go-deployd API Self-Test</h1>
        </header>

        <!-- Authentication Section -->
        <div class="auth-section">
            <h2>Authentication Configuration</h2>
            <div class="auth-grid">
                <div class="auth-method">
                    <h3>Master Key Authentication</h3>
                    <div class="form-group">
                        <label for="masterKey">Master Key</label>
                        <input type="password" id="masterKey" placeholder="Enter master key">
                    </div>
                    <button onclick="setMasterKey()">Set Master Key</button>
                    <div id="masterKeyStatus"></div>
                </div>

                <div class="auth-method">
                    <h3>JWT Authentication</h3>
                    <div class="form-group">
                        <label for="username">Username</label>
                        <input type="text" id="username" placeholder="Enter username">
                    </div>
                    <div class="form-group">
                        <label for="password">Password</label>
                        <input type="password" id="password" placeholder="Enter password">
                    </div>
                    <button onclick="login()">Login</button>
                    <button onclick="register()">Register</button>
                    <div id="jwtToken" class="token-display" style="display: none;"></div>
                </div>
            </div>
            <div style="text-align: center; margin-top: 20px;">
                Current Auth Status: <span id="authStatus" class="auth-status unauthenticated">Not Authenticated</span>
            </div>
        </div>

        <!-- Control Panel -->
        <div class="control-panel">
            <h2>Test Controls</h2>
            <div class="controls">
                <select id="collectionSelect" onchange="loadTests()">
                    <option value="">Select Collection</option>
                    <option value="users">Users</option>
                    <option value="files">Files</option>
                    <option value="custom">Custom Endpoint</option>
                </select>

                <div class="custom-endpoint" id="customEndpoint" style="display: none;">
                    <input type="text" id="customPath" placeholder="/api/endpoint">
                    <button onclick="loadTests()">Load</button>
                </div>

                <button onclick="runAllTests()" id="runAllBtn">Run All Tests</button>
                <button onclick="clearConsole()">Clear Console</button>
            </div>
        </div>

        <!-- POST Data Section -->
        <div class="post-data-section" id="postDataSection" style="display: none;">
            <h3>POST Request Data</h3>
            <div class="form-group">
                <label for="postData">JSON Data (for POST/PUT requests)</label>
                <textarea id="postData" placeholder='{"name": "John Doe", "email": "john@example.com"}'>{
  "title": "Test Item",
  "description": "This is a test item",
  "value": 123
}</textarea>
            </div>
        </div>

        <!-- File Upload Section -->
        <div class="file-section" id="fileSection" style="display: none;">
            <h3>File Management</h3>
            <div class="file-upload-area" id="dropZone">
                <p>📁 Drag and drop files here or click to select</p>
                <input type="file" id="fileInput" style="display: none;" multiple>
            </div>
            <div class="file-list" id="fileList"></div>
        </div>

        <!-- Test Grid -->
        <div class="test-grid" id="testGrid"></div>

        <!-- Console -->
        <div class="console" id="console">
            <div class="log-entry info">
                <span class="timestamp">[00:00:00]</span>
                Welcome to go-deployd API Self-Test Console
            </div>
        </div>
    </div>

    <!-- Test Details Modal -->
    <div class="modal" id="detailsModal">
        <div class="modal-content">
            <div class="modal-header">
                <h2>Test Details</h2>
                <button class="close-button" onclick="closeModal()">×</button>
            </div>
            <div id="modalContent"></div>
        </div>
    </div>

    <script>
        // Global state
        let authToken = '';
        let masterKey = '';
        let testResults = {};
        const API_BASE = '';

        // Test definitions
        const testDefinitions = {
            users: [
                {
                    name: 'GET /users',
                    method: 'GET',
                    endpoint: '/users',
                    description: 'List all users'
                },
                {
                    name: 'POST /users/register',
                    method: 'POST',
                    endpoint: '/users/register',
                    description: 'Register a new user',
                    data: {
                        username: 'testuser_' + Date.now(),
                        email: 'test' + Date.now() + '@example.com',
                        password: 'password123'
                    }
                },
                {
                    name: 'POST /users/login',
                    method: 'POST',
                    endpoint: '/users/login',
                    description: 'Login with credentials',
                    data: {
                        email: 'test@example.com',
                        password: 'password123'
                    }
                },
                {
                    name: 'GET /users/me',
                    method: 'GET',
                    endpoint: '/users/me',
                    description: 'Get current user info (requires auth)'
                }
            ],
            files: [
                {
                    name: 'GET /files',
                    method: 'GET',
                    endpoint: '/files',
                    description: 'List all files'
                },
                {
                    name: 'POST /files',
                    method: 'POST',
                    endpoint: '/files',
                    description: 'Upload a file',
                    isFileUpload: true
                },
                {
                    name: 'GET /files/:id',
                    method: 'GET',
                    endpoint: '/files/{id}',
                    description: 'Get file by ID',
                    needsId: true
                },
                {
                    name: 'DELETE /files/:id',
                    method: 'DELETE',
                    endpoint: '/files/{id}',
                    description: 'Delete file',
                    needsId: true
                }
            ]
        };

        // Authentication functions
        function setMasterKey() {
            masterKey = document.getElementById('masterKey').value;
            if (masterKey) {
                document.getElementById('masterKeyStatus').innerHTML = '<div style="color: #4CAF50; margin-top: 10px;">Master key set</div>';
                updateAuthStatus();
                log('Master key configured', 'info');
            }
        }

        async function login() {
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;

            try {
                const response = await fetch(API_BASE + '/users/login', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ username, password })
                });

                const data = await response.json();
                if (response.ok && data.token) {
                    authToken = data.token;
                    document.getElementById('jwtToken').style.display = 'block';
                    document.getElementById('jwtToken').textContent = 'Token: ' + authToken.substring(0, 20) + '...';
                    updateAuthStatus();
                    log('Login successful', 'info');
                } else {
                    log('Login failed: ' + (data.message || 'Unknown error'), 'error');
                }
            } catch (err) {
                log('Login error: ' + err.message, 'error');
            }
        }

        async function register() {
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            const email = username + '@example.com';

            try {
                const response = await fetch(API_BASE + '/users/register', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ username, email, password })
                });

                const data = await response.json();
                if (response.ok) {
                    log('Registration successful', 'info');
                    // Auto-login after registration
                    await login();
                } else {
                    log('Registration failed: ' + (data.message || 'Unknown error'), 'error');
                }
            } catch (err) {
                log('Registration error: ' + err.message, 'error');
            }
        }

        function updateAuthStatus() {
            const status = document.getElementById('authStatus');
            if (authToken || masterKey) {
                status.textContent = authToken ? 'JWT Authenticated' : 'Master Key Authenticated';
                status.className = 'auth-status authenticated';
            } else {
                status.textContent = 'Not Authenticated';
                status.className = 'auth-status unauthenticated';
            }
        }

        function getAuthHeaders() {
            const headers = {
                'Content-Type': 'application/json'
            };
            
            if (masterKey) {
                headers['X-Master-Key'] = masterKey;
            } else if (authToken) {
                headers['Authorization'] = 'Bearer ' + authToken;
            }
            
            return headers;
        }

        // Test loading and execution
        function loadTests() {
            const collection = document.getElementById('collectionSelect').value;
            const customEndpoint = document.getElementById('customEndpoint');
            const postDataSection = document.getElementById('postDataSection');
            const fileSection = document.getElementById('fileSection');
            
            // Show/hide custom endpoint input
            customEndpoint.style.display = collection === 'custom' ? 'flex' : 'none';
            
            // Show/hide sections based on collection
            postDataSection.style.display = collection && collection !== 'files' ? 'block' : 'none';
            fileSection.style.display = collection === 'files' ? 'block' : 'none';
            
            // Clear test grid
            document.getElementById('testGrid').innerHTML = '';
            
            let tests = testDefinitions[collection] || [];
            
            // If custom endpoint, create basic CRUD tests
            if (collection === 'custom') {
                const customPath = document.getElementById('customPath').value;
                if (customPath) {
                    tests = [
                        {
                            name: 'GET ' + customPath,
                            method: 'GET',
                            endpoint: customPath,
                            description: 'List all items'
                        },
                        {
                            name: 'POST ' + customPath,
                            method: 'POST',
                            endpoint: customPath,
                            description: 'Create new item',
                            data: JSON.parse(document.getElementById('postData').value || '{}')
                        }
                    ];
                }
            }
            
            // Create test cards
            tests.forEach((test, index) => {
                createTestCard(test, index);
            });
            
            log('Loaded ' + tests.length + ' tests for ' + (collection || 'custom endpoint'), 'info');
        }

        function createTestCard(test, index) {
            const card = document.createElement('div');
            card.className = 'test-card';
            card.innerHTML = '<div class="test-header">' +
                '<div class="test-title">' + test.name + '</div>' +
                '<div class="status-badge pending" id="status-' + index + '">Pending</div>' +
                '</div>' +
                '<div class="test-content">' +
                '<div class="section-title">Description</div>' +
                '<div>' + test.description + '</div>' +
                '</div>' +
                '<div class="test-actions">' +
                '<button onclick="runSingleTest(' + index + ')">Run Test</button>' +
                '<button onclick="viewTestDetails(' + index + ')">View Details</button>' +
                '</div>' +
                '<div class="progress-bar">' +
                '<div class="progress-fill" id="progress-' + index + '" style="width: 0%"></div>' +
                '</div>';
            document.getElementById('testGrid').appendChild(card);
        }

        async function runAllTests() {
            const collection = document.getElementById('collectionSelect').value;
            if (!collection) {
                log('Please select a collection first', 'warning');
                return;
            }

            const runBtn = document.getElementById('runAllBtn');
            runBtn.disabled = true;
            runBtn.textContent = 'Running...';

            log('Starting test suite for ' + collection, 'info');

            const tests = testDefinitions[collection] || [];
            let createdId = null;

            for (let i = 0; i < tests.length; i++) {
                const test = tests[i];
                
                // If test needs ID and we have one from creation, use it
                if (test.needsId && createdId) {
                    test.endpoint = test.endpoint.replace('{id}', createdId);
                }
                
                const result = await runTest(test, i);
                
                // Store created ID for subsequent tests
                if (test.method === 'POST' && result.success && result.data && result.data.id) {
                    createdId = result.data.id;
                }
            }

            runBtn.disabled = false;
            runBtn.textContent = 'Run All Tests';
            log('Test suite completed', 'info');
        }

        async function runSingleTest(index) {
            const collection = document.getElementById('collectionSelect').value;
            const tests = testDefinitions[collection] || [];
            await runTest(tests[index], index);
        }

        async function runTest(test, index) {
            const statusEl = document.getElementById('status-' + index);
            const progressEl = document.getElementById('progress-' + index);
            
            statusEl.textContent = 'Running';
            statusEl.className = 'status-badge warning';
            progressEl.style.width = '50%';

            const result = {
                test: test,
                success: false,
                status: null,
                data: null,
                error: null,
                responseTime: 0
            };

            const startTime = Date.now();

            try {
                let response;
                
                if (test.isFileUpload) {
                    // Handle file upload
                    const formData = new FormData();
                    const fileInput = document.getElementById('fileInput');
                    if (fileInput.files.length > 0) {
                        formData.append('file', fileInput.files[0]);
                    } else {
                        // Create a test file
                        const blob = new Blob(['Test file content'], { type: 'text/plain' });
                        formData.append('file', blob, 'test.txt');
                    }
                    
                    const headers = {};
                    if (masterKey) headers['X-Master-Key'] = masterKey;
                    if (authToken) headers['Authorization'] = 'Bearer ' + authToken;
                    
                    response = await fetch(API_BASE + test.endpoint, {
                        method: test.method,
                        headers: headers,
                        body: formData
                    });
                } else {
                    // Regular request
                    const options = {
                        method: test.method,
                        headers: getAuthHeaders()
                    };
                    
                    if (test.data || (test.method === 'POST' && document.getElementById('postData'))) {
                        options.body = JSON.stringify(
                            test.data || JSON.parse(document.getElementById('postData').value || '{}')
                        );
                    }
                    
                    response = await fetch(API_BASE + test.endpoint, options);
                }

                const responseTime = Date.now() - startTime;
                result.responseTime = responseTime;
                result.status = response.status;
                
                try {
                    result.data = await response.json();
                } catch (e) {
                    result.data = await response.text();
                }
                
                result.success = response.ok;
                
                if (response.ok) {
                    statusEl.textContent = 'Success';
                    statusEl.className = 'status-badge success';
                    log('✅ ' + test.name + ' - ' + responseTime + 'ms', 'info');
                } else {
                    statusEl.textContent = 'Error';
                    statusEl.className = 'status-badge error';
                    log('❌ ' + test.name + ' - Status: ' + response.status, 'error');
                }
            } catch (err) {
                result.success = false;
                result.error = err.message;
                result.responseTime = Date.now() - startTime;
                
                statusEl.textContent = 'Error';
                statusEl.className = 'status-badge error';
                log('❌ ' + test.name + ' - ' + err.message, 'error');
            }
            
            progressEl.style.width = '100%';
            testResults[index] = result;
            
            return result;
        }

        function viewTestDetails(index) {
            const result = testResults[index];
            if (!result) {
                log('No test results available', 'warning');
                return;
            }

            const modal = document.getElementById('detailsModal');
            const content = document.getElementById('modalContent');
            
            const requestData = result.test.data ? JSON.stringify(result.test.data, null, 2) : '';
            const responseData = typeof result.data === 'object' ? 
                JSON.stringify(result.data, null, 2) : result.data;
            
            content.innerHTML = '<div class="request-section">' +
                '<div class="section-title">Request</div>' +
                '<div class="code-block">' +
                result.test.method + ' ' + result.test.endpoint + '\n' +
                'Headers: ' + JSON.stringify(getAuthHeaders(), null, 2) + '\n' +
                (requestData ? 'Body: ' + requestData : '') +
                '</div>' +
                '</div>' +
                '<div class="response-section">' +
                '<div class="section-title">Response</div>' +
                '<div class="code-block">' +
                'Status: ' + (result.status || 'N/A') + '\n' +
                'Time: ' + result.responseTime + 'ms\n\n' +
                responseData +
                '</div>' +
                '</div>';
            
            modal.style.display = 'block';
        }

        function closeModal() {
            document.getElementById('detailsModal').style.display = 'none';
        }

        // File handling
        function setupFileHandling() {
            const dropZone = document.getElementById('dropZone');
            const fileInput = document.getElementById('fileInput');

            dropZone.addEventListener('click', () => fileInput.click());
            
            dropZone.addEventListener('dragover', (e) => {
                e.preventDefault();
                dropZone.classList.add('dragover');
            });
            
            dropZone.addEventListener('dragleave', () => {
                dropZone.classList.remove('dragover');
            });
            
            dropZone.addEventListener('drop', (e) => {
                e.preventDefault();
                dropZone.classList.remove('dragover');
                handleFiles(e.dataTransfer.files);
            });
            
            fileInput.addEventListener('change', (e) => {
                handleFiles(e.target.files);
            });
        }

        async function handleFiles(files) {
            for (const file of files) {
                await uploadFile(file);
            }
            await loadFileList();
        }

        async function uploadFile(file) {
            const formData = new FormData();
            formData.append('file', file);
            
            const headers = {};
            if (masterKey) headers['X-Master-Key'] = masterKey;
            if (authToken) headers['Authorization'] = 'Bearer ' + authToken;
            
            try {
                const response = await fetch(API_BASE + '/files', {
                    method: 'POST',
                    headers: headers,
                    body: formData
                });
                
                if (response.ok) {
                    log('File uploaded: ' + file.name, 'info');
                } else {
                    log('File upload failed: ' + file.name, 'error');
                }
            } catch (err) {
                log('Upload error: ' + err.message, 'error');
            }
        }

        async function loadFileList() {
            try {
                const response = await fetch(API_BASE + '/files', {
                    headers: getAuthHeaders()
                });
                
                if (response.ok) {
                    const files = await response.json();
                    displayFileList(files);
                }
            } catch (err) {
                log('Failed to load file list: ' + err.message, 'error');
            }
        }

        function displayFileList(files) {
            const fileList = document.getElementById('fileList');
            fileList.innerHTML = '';
            
            files.forEach(file => {
                const item = document.createElement('div');
                item.className = 'file-item';
                item.innerHTML = '<div class="file-info">' +
                    '<div class="file-name">' + (file.originalName || file.filename) + '</div>' +
                    '<div class="file-details">' +
                    formatFileSize(file.size) + ' • ' + file.contentType + ' • ' + new Date(file.uploadedAt).toLocaleString() +
                    '</div>' +
                    '</div>' +
                    '<div class="file-actions">' +
                    '<button onclick="downloadFile(\'' + file.id + '\', \'' + (file.originalName || file.filename) + '\')">📥 Download</button>' +
                    '<button onclick="deleteFile(\'' + file.id + '\')" style="background: #f44336;">🗑️ Delete</button>' +
                    '</div>';
                fileList.appendChild(item);
            });
            
            if (files.length === 0) {
                fileList.innerHTML = '<div style="text-align: center; color: #666;">No files uploaded yet</div>';
            }
        }

        async function downloadFile(id, filename) {
            try {
                const response = await fetch(API_BASE + '/files/' + id, {
                    headers: getAuthHeaders()
                });
                
                if (response.ok) {
                    const blob = await response.blob();
                    const url = window.URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = filename;
                    a.click();
                    window.URL.revokeObjectURL(url);
                    log('File downloaded: ' + filename, 'info');
                } else {
                    log('Download failed', 'error');
                }
            } catch (err) {
                log('Download error: ' + err.message, 'error');
            }
        }

        async function deleteFile(id) {
            if (!confirm('Are you sure you want to delete this file?')) return;
            
            try {
                const response = await fetch(API_BASE + '/files/' + id, {
                    method: 'DELETE',
                    headers: getAuthHeaders()
                });
                
                if (response.ok) {
                    log('File deleted', 'info');
                    await loadFileList();
                } else {
                    log('Delete failed', 'error');
                }
            } catch (err) {
                log('Delete error: ' + err.message, 'error');
            }
        }

        // Utility functions
        function log(message, type = 'info') {
            const console = document.getElementById('console');
            const entry = document.createElement('div');
            entry.className = 'log-entry ' + type;
            
            const timestamp = new Date().toLocaleTimeString();
            entry.innerHTML = '<span class="timestamp">[' + timestamp + ']</span>' + message;
            
            console.appendChild(entry);
            console.scrollTop = console.scrollHeight;
        }

        function clearConsole() {
            document.getElementById('console').innerHTML = '';
            log('Console cleared', 'info');
        }

        function formatFileSize(bytes) {
            if (bytes === 0) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        // Initialize
        document.addEventListener('DOMContentLoaded', () => {
            setupFileHandling();
            
            // Load files if file collection is selected
            document.getElementById('collectionSelect').addEventListener('change', (e) => {
                if (e.target.value === 'files') {
                    loadFileList();
                }
            });
            
            // Close modal on outside click
            document.getElementById('detailsModal').addEventListener('click', (e) => {
                if (e.target.id === 'detailsModal') {
                    closeModal();
                }
            });
            
            log('Self-test interface ready', 'info');
        });
    </script>
</body>
</html>`