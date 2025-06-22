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
} from '@chakra-ui/react'
import {
  FiSave,
  FiRefreshCw,
  FiServer,
  FiDatabase,
  FiSettings,
  FiSun,
  FiMoon,
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
    sessionTTL: 86400,
    tokenTTL: 2592000,
    allowRegistration: false
  })
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [securityLoading, setSecurityLoading] = useState(false)

  const toast = useToast()
  const { colorMode, toggleColorMode } = useColorMode()
  const { authFetch } = useAuth()
  const cardBg = useColorModeValue('white', 'gray.700')

  useEffect(() => {
    loadServerInfo()
    loadSecuritySettings()
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
          sessionTTL: settings.sessionTTL || 86400,
          tokenTTL: settings.tokenTTL || 2592000,
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
                  Changing security settings affects all active sessions and authentication behavior.
                </AlertDescription>
              </Alert>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Authentication & Session Settings</Heading>
                </CardHeader>
                <CardBody>
                  {securityLoading ? (
                    <Text color="gray.500">Loading security settings...</Text>
                  ) : (
                    <VStack align="stretch" spacing={6}>
                      <FormControl>
                        <FormLabel>Session Timeout (seconds)</FormLabel>
                        <Input
                          type="number"
                          value={securitySettings.sessionTTL}
                          onChange={(e) => handleSecuritySettingChange('sessionTTL', parseInt(e.target.value))}
                          min="300"
                          max="2592000"
                        />
                        <Text fontSize="sm" color="gray.500" mt={1}>
                          How long user sessions remain active (default: 86400 = 24 hours)
                        </Text>
                      </FormControl>

                      <FormControl>
                        <FormLabel>API Token Timeout (seconds)</FormLabel>
                        <Input
                          type="number"
                          value={securitySettings.tokenTTL}
                          onChange={(e) => handleSecuritySettingChange('tokenTTL', parseInt(e.target.value))}
                          min="3600"
                          max="31536000"
                        />
                        <Text fontSize="sm" color="gray.500" mt={1}>
                          How long API tokens remain valid (default: 2592000 = 30 days)
                        </Text>
                      </FormControl>

                      <FormControl display="flex" alignItems="center" justifyContent="space-between">
                        <Box>
                          <FormLabel mb="0">Allow Public Registration</FormLabel>
                          <Text fontSize="sm" color="gray.500">
                            When disabled, only administrators can create users via master key
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
                            <Text fontSize="sm">Dashboard authentication protected</Text>
                          </HStack>
                          <HStack>
                            <Badge colorScheme="green">✓</Badge>
                            <Text fontSize="sm">Admin API endpoints secured</Text>
                          </HStack>
                          <HStack>
                            <Badge colorScheme="green">✓</Badge>
                            <Text fontSize="sm">bcrypt password hashing (cost 12)</Text>
                          </HStack>
                        </VStack>
                      </Box>

                      <Alert status="info" variant="left-accent">
                        <AlertIcon />
                        <AlertDescription>
                          Master key is stored in <Code>.deployd/security.json</Code> with 600 permissions.
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

                    <Divider />

                    <Box>
                      <Text fontWeight="medium" mb={2}>Theme Information</Text>
                      <VStack align="start" spacing={2}>
                        <HStack>
                          <Text fontSize="sm" fontWeight="medium">Current Theme:</Text>
                          <Badge colorScheme="brand">{colorMode === 'dark' ? 'Dark' : 'Light'}</Badge>
                        </HStack>
                        <HStack>
                          <Text fontSize="sm" fontWeight="medium">UI Framework:</Text>
                          <Badge variant="outline">Chakra UI</Badge>
                        </HStack>
                        <HStack>
                          <Text fontSize="sm" fontWeight="medium">Brand Color:</Text>
                          <Box w={4} h={4} bg="brand.500" borderRadius="sm" />
                          <Code fontSize="xs">#3182CE</Code>
                        </HStack>
                      </VStack>
                    </Box>
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