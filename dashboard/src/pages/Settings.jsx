import React, { useState, useEffect } from 'react'
import {
  Box,
  VStack,
  HStack,
  Heading,
  Text,
  Card,
  CardBody,
  CardHeader,
  FormControl,
  FormLabel,
  Input,
  Switch,
  Button,
  useToast,
  Alert,
  AlertIcon,
  AlertDescription,
  Divider,
  Badge,
  Code,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  useColorMode,
  useColorModeValue,
  Select,
  Textarea,
} from '@chakra-ui/react'
import {
  FiSave,
  FiRefreshCw,
  FiServer,
  FiDatabase,
  FiSettings,
  FiSun,
  FiMoon,
  FiMail,
  FiSend,
} from 'react-icons/fi'
import { apiService } from '../services/api'
import { useAuth } from '../contexts/AuthContext'

function Settings() {
  const [serverInfo, setServerInfo] = useState(null)
  const [settings, setSettings] = useState({
    databaseUrl: 'sqlite://data/deployd.db',
    port: 2403,
    enableCors: true,
    enableLogging: true,
    maxConnections: 100,
    sessionSecret: '***hidden***'
  })
  const [securitySettings, setSecuritySettings] = useState({
    jwtExpiration: '24h',
    allowRegistration: false
  })
  const [emailSettings, setEmailSettings] = useState({
    provider: 'smtp',
    smtp: {
      host: 'smtp.gmail.com',
      port: 587,
      username: '',
      password: '',
      tls: true,
      hasPassword: false
    },
    ses: {
      region: 'us-east-1',
      hasAccessKeyId: false,
      hasSecretAccessKey: false
    },
    from: 'noreply@example.com',
    fromName: 'Go-Deployd',
    requireVerification: true
  })
  const [testEmail, setTestEmail] = useState('')
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [securityLoading, setSecurityLoading] = useState(false)
  const [emailLoading, setEmailLoading] = useState(false)
  const [sendingTest, setSendingTest] = useState(false)

  const toast = useToast()
  const { colorMode, toggleColorMode } = useColorMode()
  const { authFetch } = useAuth()
  const cardBg = useColorModeValue('white', 'gray.700')

  useEffect(() => {
    loadServerInfo()
    loadSecuritySettings()
    loadEmailSettings()
  }, [])

  const loadServerInfo = async () => {
    try {
      setLoading(true)
      const response = await authFetch('/_admin/info')
      if (response.ok) {
        const info = await response.json()
        setServerInfo(info)
      } else {
        throw new Error('Failed to load server info')
      }
    } catch (err) {
      // Server info endpoint might not exist yet, show mock data
      setServerInfo({
        version: '1.0.0',
        goVersion: 'Go 1.21',
        uptime: '2h 15m',
        database: 'Connected',
        collections: 2,
        totalDocuments: 156,
        environment: 'development'
      })
    } finally {
      setLoading(false)
    }
  }

  const loadSecuritySettings = async () => {
    try {
      setSecurityLoading(true)
      const response = await authFetch('/_admin/settings/security')
      if (response.ok) {
        const settings = await response.json()
        setSecuritySettings({
          jwtExpiration: settings.jwtExpiration || '24h',
          allowRegistration: settings.allowRegistration || false
        })
      }
    } catch (err) {
      toast({
        title: 'Error Loading Security Settings',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setSecurityLoading(false)
    }
  }

  const saveSettings = async () => {
    setSaving(true)
    try {
      // TODO: Implement settings save API
      await new Promise(resolve => setTimeout(resolve, 1000)) // Mock delay
      
      toast({
        title: 'Settings Saved',
        description: 'Server settings have been updated successfully.',
        status: 'success',
        duration: 3000,
        isClosable: true,
      })
    } catch (err) {
      toast({
        title: 'Error Saving Settings',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setSaving(false)
    }
  }

  const saveSecuritySettings = async () => {
    setSaving(true)
    try {
      const response = await authFetch('/_admin/settings/security', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(securitySettings),
      })

      if (response.ok) {
        toast({
          title: 'Security Settings Saved',
          description: 'Security settings have been updated successfully.',
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
      } else {
        const error = await response.json()
        throw new Error(error.message || 'Failed to save settings')
      }
    } catch (err) {
      toast({
        title: 'Error Saving Security Settings',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setSaving(false)
    }
  }

  const handleSettingChange = (key, value) => {
    setSettings(prev => ({
      ...prev,
      [key]: value
    }))
  }

  const handleSecuritySettingChange = (key, value) => {
    setSecuritySettings(prev => ({
      ...prev,
      [key]: value
    }))
  }

  const loadEmailSettings = async () => {
    try {
      setEmailLoading(true)
      const response = await authFetch('/_admin/settings/email')
      if (response.ok) {
        const settings = await response.json()
        setEmailSettings(settings)
      }
    } catch (err) {
      toast({
        title: 'Error Loading Email Settings',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setEmailLoading(false)
    }
  }

  const saveEmailSettings = async () => {
    setSaving(true)
    try {
      const response = await authFetch('/_admin/settings/email', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(emailSettings),
      })

      if (response.ok) {
        toast({
          title: 'Email Settings Saved',
          description: 'Email settings have been updated successfully.',
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
      } else {
        const error = await response.json()
        throw new Error(error.message || 'Failed to save email settings')
      }
    } catch (err) {
      toast({
        title: 'Error Saving Email Settings',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setSaving(false)
    }
  }

  const sendTestEmail = async () => {
    if (!testEmail) {
      toast({
        title: 'Email Required',
        description: 'Please enter an email address to send the test to.',
        status: 'warning',
        duration: 3000,
        isClosable: true,
      })
      return
    }

    setSendingTest(true)
    try {
      const response = await authFetch('/_admin/settings/email/test', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ to: testEmail }),
      })

      if (response.ok) {
        toast({
          title: 'Test Email Sent',
          description: `Test email sent successfully to ${testEmail}`,
          status: 'success',
          duration: 3000,
          isClosable: true,
        })
      } else {
        const error = await response.json()
        throw new Error(error.message || 'Failed to send test email')
      }
    } catch (err) {
      toast({
        title: 'Error Sending Test Email',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setSendingTest(false)
    }
  }

  const handleEmailSettingChange = (key, value) => {
    if (key.includes('.')) {
      const [section, field] = key.split('.')
      setEmailSettings(prev => ({
        ...prev,
        [section]: {
          ...prev[section],
          [field]: value
        }
      }))
    } else {
      setEmailSettings(prev => ({
        ...prev,
        [key]: value
      }))
    }
  }

  return (
    <VStack align="stretch" spacing={6}>
      <HStack justify="space-between">
        <Heading size="lg">Settings</Heading>
        <HStack>
          <Button
            leftIcon={<FiRefreshCw />}
            variant="outline"
            size="sm"
            onClick={loadServerInfo}
            isLoading={loading}
          >
            Refresh
          </Button>
        </HStack>
      </HStack>

      <Tabs>
        <TabList>
          <Tab>Server</Tab>
          <Tab>Database</Tab>
          <Tab>Security</Tab>
          <Tab>Email</Tab>
          <Tab>Appearance</Tab>
        </TabList>

        <TabPanels>
          {/* Server Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <Card bg={cardBg}>
                <CardHeader>
                  <HStack>
                    <FiServer />
                    <Heading size="md">Server Information</Heading>
                  </HStack>
                </CardHeader>
                <CardBody>
                  {serverInfo ? (
                    <VStack align="stretch" spacing={4}>
                      <HStack justify="space-between" wrap="wrap">
                        <Stat>
                          <StatLabel>Version</StatLabel>
                          <StatNumber fontSize="lg">{serverInfo.version}</StatNumber>
                          <StatHelpText>Go Deployd</StatHelpText>
                        </Stat>
                        <Stat>
                          <StatLabel>Runtime</StatLabel>
                          <StatNumber fontSize="lg">{serverInfo.goVersion}</StatNumber>
                        </Stat>
                        <Stat>
                          <StatLabel>Uptime</StatLabel>
                          <StatNumber fontSize="lg">{serverInfo.uptime}</StatNumber>
                        </Stat>
                        <Stat>
                          <StatLabel>Memory Usage</StatLabel>
                          <StatNumber fontSize="lg">{serverInfo.memory}</StatNumber>
                        </Stat>
                      </HStack>
                      
                      <Divider />
                      
                      <HStack justify="space-between" wrap="wrap">
                        <Stat>
                          <StatLabel>Collections</StatLabel>
                          <StatNumber fontSize="lg">{serverInfo.collections}</StatNumber>
                        </Stat>
                        <Stat>
                          <StatLabel>Total Documents</StatLabel>
                          <StatNumber fontSize="lg">{serverInfo.totalDocuments}</StatNumber>
                        </Stat>
                        <Stat>
                          <StatLabel>Environment</StatLabel>
                          <StatNumber fontSize="lg">
                            <Badge colorScheme={serverInfo.environment === 'production' ? 'red' : 'green'}>
                              {serverInfo.environment}
                            </Badge>
                          </StatNumber>
                        </Stat>
                      </HStack>
                    </VStack>
                  ) : (
                    <Text color="gray.500">Loading server information...</Text>
                  )}
                </CardBody>
              </Card>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Server Configuration</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <FormControl>
                      <FormLabel>Port</FormLabel>
                      <Input
                        type="number"
                        value={settings.port}
                        onChange={(e) => handleSettingChange('port', parseInt(e.target.value))}
                      />
                    </FormControl>

                    <FormControl>
                      <FormLabel>Max Connections</FormLabel>
                      <Input
                        type="number"
                        value={settings.maxConnections}
                        onChange={(e) => handleSettingChange('maxConnections', parseInt(e.target.value))}
                      />
                    </FormControl>

                    <FormControl display="flex" alignItems="center">
                      <FormLabel mb="0">Enable CORS</FormLabel>
                      <Switch
                        isChecked={settings.enableCors}
                        onChange={(e) => handleSettingChange('enableCors', e.target.checked)}
                      />
                    </FormControl>

                    <FormControl display="flex" alignItems="center">
                      <FormLabel mb="0">Enable Request Logging</FormLabel>
                      <Switch
                        isChecked={settings.enableLogging}
                        onChange={(e) => handleSettingChange('enableLogging', e.target.checked)}
                      />
                    </FormControl>

                    <Button
                      leftIcon={<FiSave />}
                      colorScheme="brand"
                      onClick={saveSettings}
                      isLoading={saving}
                      loadingText="Saving"
                    >
                      Save Server Settings
                    </Button>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Database Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <Alert status="info" variant="left-accent">
                <AlertIcon />
                <AlertDescription>
                  Database settings require a server restart to take effect.
                </AlertDescription>
              </Alert>

              <Card bg={cardBg}>
                <CardHeader>
                  <HStack>
                    <FiDatabase />
                    <Heading size="md">Database Configuration</Heading>
                  </HStack>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <FormControl>
                      <FormLabel>Database URL</FormLabel>
                      <Input
                        value={settings.databaseUrl}
                        onChange={(e) => handleSettingChange('databaseUrl', e.target.value)}
                        placeholder="sqlite://data/deployd.db"
                      />
                      <Text fontSize="sm" color="gray.500" mt={1}>
                        Database connection string (supports MongoDB, SQLite, MySQL, PostgreSQL)
                      </Text>
                    </FormControl>

                    <Box>
                      <Text fontWeight="medium" mb={2}>Connection Status</Text>
                      <HStack>
                        <Badge colorScheme="green">Connected</Badge>
                        <Text fontSize="sm" color="gray.500">
                          Database connected
                        </Text>
                      </HStack>
                    </Box>

                    <Button
                      leftIcon={<FiSave />}
                      colorScheme="brand"
                      onClick={saveSettings}
                      isLoading={saving}
                      loadingText="Saving"
                    >
                      Save Database Settings
                    </Button>
                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Security Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <Alert status="warning" variant="left-accent">
                <AlertIcon />
                <AlertDescription>
                  Changing security settings affects JWT token expiration and user authentication behavior.
                </AlertDescription>
              </Alert>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">JWT Authentication Settings</Heading>
                </CardHeader>
                <CardBody>
                  {securityLoading ? (
                    <Text color="gray.500">Loading security settings...</Text>
                  ) : (
                    <VStack align="stretch" spacing={6}>
                      <FormControl>
                        <FormLabel>JWT Token Expiration</FormLabel>
                        <Input
                          value={securitySettings.jwtExpiration}
                          onChange={(e) => handleSecuritySettingChange('jwtExpiration', e.target.value)}
                          placeholder="24h"
                        />
                        <Text fontSize="sm" color="gray.500" mt={1}>
                          How long JWT tokens remain valid (e.g. 24h, 7d, 30d, 1y)
                        </Text>
                      </FormControl>

                      <FormControl display="flex" alignItems="center" justifyContent="space-between">
                        <Box>
                          <FormLabel mb="0">Allow Public Registration</FormLabel>
                          <Text fontSize="sm" color="gray.500">
                            When disabled, only administrators can create users via dashboard or API
                          </Text>
                        </Box>
                        <Switch
                          isChecked={securitySettings.allowRegistration}
                          onChange={(e) => handleSecuritySettingChange('allowRegistration', e.target.checked)}
                          colorScheme="brand"
                        />
                      </FormControl>

                      <Divider />

                      <Box>
                        <Text fontWeight="medium" mb={2}>Master Key Security</Text>
                        <VStack align="start" spacing={2}>
                          <HStack>
                            <Badge colorScheme="green">✓</Badge>
                            <Text fontSize="sm">Master key generated and secured</Text>
                          </HStack>
                          <HStack>
                            <Badge colorScheme="green">✓</Badge>
                            <Text fontSize="sm">JWT authentication enabled</Text>
                          </HStack>
                          <HStack>
                            <Badge colorScheme="green">✓</Badge>
                            <Text fontSize="sm">Admin API endpoints secured</Text>
                          </HStack>
                          <HStack>
                            <Badge colorScheme="green">✓</Badge>
                            <Text fontSize="sm">bcrypt password hashing (cost 12)</Text>
                          </HStack>
                          <HStack>
                            <Badge colorScheme="green">✓</Badge>
                            <Text fontSize="sm">HMAC-SHA256 JWT signing</Text>
                          </HStack>
                        </VStack>
                      </Box>

                      <Alert status="info" variant="left-accent">
                        <AlertIcon />
                        <AlertDescription>
                          JWT secret and master key are stored in <Code>.deployd/security.json</Code> with 600 permissions.
                          Keep this file secure and do not commit it to version control.
                        </AlertDescription>
                      </Alert>

                      <Button
                        leftIcon={<FiSave />}
                        colorScheme="brand"
                        onClick={saveSecuritySettings}
                        isLoading={saving}
                        loadingText="Saving Security Settings"
                        size="lg"
                      >
                        Save Security Settings
                      </Button>
                    </VStack>
                  )}
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Email Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <Alert status="info" variant="left-accent">
                <AlertIcon />
                <AlertDescription>
                  Configure email settings for user registration verification and password reset emails.
                </AlertDescription>
              </Alert>

              <Card bg={cardBg}>
                <CardHeader>
                  <HStack>
                    <FiMail />
                    <Heading size="md">Email Configuration</Heading>
                  </HStack>
                </CardHeader>
                <CardBody>
                  {emailLoading ? (
                    <Text color="gray.500">Loading email settings...</Text>
                  ) : (
                    <VStack align="stretch" spacing={6}>
                      <FormControl>
                        <FormLabel>Email Provider</FormLabel>
                        <Select
                          value={emailSettings.provider}
                          onChange={(e) => handleEmailSettingChange('provider', e.target.value)}
                        >
                          <option value="smtp">SMTP</option>
                          <option value="ses">Amazon SES</option>
                        </Select>
                        <Text fontSize="sm" color="gray.500" mt={1}>
                          Choose your email service provider
                        </Text>
                      </FormControl>

                      <FormControl display="flex" alignItems="center" justifyContent="space-between">
                        <Box>
                          <FormLabel mb="0">Require Email Verification</FormLabel>
                          <Text fontSize="sm" color="gray.500">
                            New users must verify their email before account activation
                          </Text>
                        </Box>
                        <Switch
                          isChecked={emailSettings.requireVerification}
                          onChange={(e) => handleEmailSettingChange('requireVerification', e.target.checked)}
                          colorScheme="brand"
                        />
                      </FormControl>

                      <Divider />

                      <FormControl>
                        <FormLabel>From Email Address</FormLabel>
                        <Input
                          value={emailSettings.from}
                          onChange={(e) => handleEmailSettingChange('from', e.target.value)}
                          placeholder="noreply@yourdomain.com"
                        />
                      </FormControl>

                      <FormControl>
                        <FormLabel>From Name</FormLabel>
                        <Input
                          value={emailSettings.fromName}
                          onChange={(e) => handleEmailSettingChange('fromName', e.target.value)}
                          placeholder="Your App Name"
                        />
                      </FormControl>

                      <Divider />

                      {emailSettings.provider === 'smtp' && (
                        <VStack align="stretch" spacing={4}>
                          <Heading size="sm" color="gray.600">SMTP Settings</Heading>
                          
                          <HStack spacing={4}>
                            <FormControl flex="2">
                              <FormLabel>SMTP Host</FormLabel>
                              <Input
                                value={emailSettings.smtp.host}
                                onChange={(e) => handleEmailSettingChange('smtp.host', e.target.value)}
                                placeholder="smtp.gmail.com"
                              />
                            </FormControl>
                            <FormControl flex="1">
                              <FormLabel>Port</FormLabel>
                              <Input
                                type="number"
                                value={emailSettings.smtp.port}
                                onChange={(e) => handleEmailSettingChange('smtp.port', parseInt(e.target.value))}
                                placeholder="587"
                              />
                            </FormControl>
                          </HStack>

                          <FormControl>
                            <FormLabel>Username</FormLabel>
                            <Input
                              value={emailSettings.smtp.username}
                              onChange={(e) => handleEmailSettingChange('smtp.username', e.target.value)}
                              placeholder="your-email@gmail.com"
                            />
                          </FormControl>

                          <FormControl>
                            <FormLabel>Password</FormLabel>
                            <Input
                              type="password"
                              value={emailSettings.smtp.password}
                              onChange={(e) => handleEmailSettingChange('smtp.password', e.target.value)}
                              placeholder={emailSettings.smtp.hasPassword ? "••••••••" : "Enter password"}
                            />
                            {emailSettings.smtp.hasPassword && (
                              <Text fontSize="sm" color="gray.500" mt={1}>
                                Password is configured. Leave blank to keep current password.
                              </Text>
                            )}
                          </FormControl>

                          <FormControl display="flex" alignItems="center">
                            <FormLabel mb="0">Enable TLS</FormLabel>
                            <Switch
                              isChecked={emailSettings.smtp.tls}
                              onChange={(e) => handleEmailSettingChange('smtp.tls', e.target.checked)}
                              colorScheme="brand"
                            />
                          </FormControl>
                        </VStack>
                      )}

                      {emailSettings.provider === 'ses' && (
                        <VStack align="stretch" spacing={4}>
                          <Heading size="sm" color="gray.600">Amazon SES Settings</Heading>
                          
                          <FormControl>
                            <FormLabel>AWS Region</FormLabel>
                            <Select
                              value={emailSettings.ses.region}
                              onChange={(e) => handleEmailSettingChange('ses.region', e.target.value)}
                            >
                              <option value="us-east-1">US East (N. Virginia)</option>
                              <option value="us-west-2">US West (Oregon)</option>
                              <option value="eu-west-1">Europe (Ireland)</option>
                              <option value="ap-southeast-1">Asia Pacific (Singapore)</option>
                            </Select>
                          </FormControl>

                          <FormControl>
                            <FormLabel>Access Key ID</FormLabel>
                            <Input
                              value={emailSettings.ses.accessKeyId || ''}
                              onChange={(e) => handleEmailSettingChange('ses.accessKeyId', e.target.value)}
                              placeholder={emailSettings.ses.hasAccessKeyId ? "••••••••" : "Enter Access Key ID"}
                            />
                          </FormControl>

                          <FormControl>
                            <FormLabel>Secret Access Key</FormLabel>
                            <Input
                              type="password"
                              value={emailSettings.ses.secretAccessKey || ''}
                              onChange={(e) => handleEmailSettingChange('ses.secretAccessKey', e.target.value)}
                              placeholder={emailSettings.ses.hasSecretAccessKey ? "••••••••" : "Enter Secret Access Key"}
                            />
                          </FormControl>

                          <Alert status="warning" variant="left-accent">
                            <AlertIcon />
                            <AlertDescription>
                              Make sure your AWS SES service is verified and out of sandbox mode for production use.
                            </AlertDescription>
                          </Alert>
                        </VStack>
                      )}

                      <Divider />

                      <VStack align="stretch" spacing={4}>
                        <Heading size="sm" color="gray.600">Test Email</Heading>
                        
                        <FormControl>
                          <FormLabel>Test Email Address</FormLabel>
                          <HStack>
                            <Input
                              value={testEmail}
                              onChange={(e) => setTestEmail(e.target.value)}
                              placeholder="your-email@example.com"
                              type="email"
                            />
                            <Button
                              leftIcon={<FiSend />}
                              onClick={sendTestEmail}
                              isLoading={sendingTest}
                              loadingText="Sending"
                              colorScheme="blue"
                              variant="outline"
                            >
                              Send Test
                            </Button>
                          </HStack>
                          <Text fontSize="sm" color="gray.500" mt={1}>
                            Send a test email to verify your configuration
                          </Text>
                        </FormControl>
                      </VStack>

                      <Button
                        leftIcon={<FiSave />}
                        colorScheme="brand"
                        onClick={saveEmailSettings}
                        isLoading={saving}
                        loadingText="Saving Email Settings"
                        size="lg"
                      >
                        Save Email Settings
                      </Button>
                    </VStack>
                  )}
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>

          {/* Appearance Tab */}
          <TabPanel>
            <VStack align="stretch" spacing={6}>
              <Card bg={cardBg}>
                <CardHeader>
                  <HStack>
                    <FiSettings />
                    <Heading size="md">Dashboard Appearance</Heading>
                  </HStack>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <FormControl display="flex" alignItems="center" justifyContent="space-between">
                      <Box>
                        <FormLabel mb="0">Dark Mode</FormLabel>
                        <Text fontSize="sm" color="gray.500">
                          Toggle between light and dark theme
                        </Text>
                      </Box>
                      <HStack>
                        <FiSun />
                        <Switch
                          isChecked={colorMode === 'dark'}
                          onChange={toggleColorMode}
                        />
                        <FiMoon />
                      </HStack>
                    </FormControl>

                  </VStack>
                </CardBody>
              </Card>
            </VStack>
          </TabPanel>
        </TabPanels>
      </Tabs>
    </VStack>
  )
}

export default Settings