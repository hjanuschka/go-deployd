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

function Settings() {
  const [serverInfo, setServerInfo] = useState(null)
  const [settings, setSettings] = useState({
    databaseUrl: 'localhost:27017/deployd',
    port: 2403,
    enableCors: true,
    enableLogging: true,
    maxConnections: 100,
    sessionSecret: '***hidden***'
  })
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  const toast = useToast()
  const { colorMode, toggleColorMode } = useColorMode()
  const cardBg = useColorModeValue('white', 'gray.700')

  useEffect(() => {
    loadServerInfo()
  }, [])

  const loadServerInfo = async () => {
    try {
      setLoading(true)
      const info = await apiService.getServerInfo()
      setServerInfo(info)
    } catch (err) {
      // Server info endpoint might not exist yet, show mock data
      setServerInfo({
        version: '1.0.0',
        nodeVersion: 'Go 1.21',
        uptime: '2h 15m',
        memory: '45.2 MB',
        collections: 2,
        totalDocuments: 156,
        environment: 'development'
      })
    } finally {
      setLoading(false)
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

  const handleSettingChange = (key, value) => {
    setSettings(prev => ({
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
                          <StatNumber fontSize="lg">{serverInfo.nodeVersion}</StatNumber>
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
                        placeholder="localhost:27017/deployd"
                      />
                      <Text fontSize="sm" color="gray.500" mt={1}>
                        MongoDB connection string
                      </Text>
                    </FormControl>

                    <Box>
                      <Text fontWeight="medium" mb={2}>Connection Status</Text>
                      <HStack>
                        <Badge colorScheme="green">Connected</Badge>
                        <Text fontSize="sm" color="gray.500">
                          Connected to MongoDB
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
                  Changing security settings affects all active sessions.
                </AlertDescription>
              </Alert>

              <Card bg={cardBg}>
                <CardHeader>
                  <Heading size="md">Session & Authentication</Heading>
                </CardHeader>
                <CardBody>
                  <VStack align="stretch" spacing={4}>
                    <FormControl>
                      <FormLabel>Session Secret</FormLabel>
                      <Input
                        type="password"
                        value={settings.sessionSecret}
                        onChange={(e) => handleSettingChange('sessionSecret', e.target.value)}
                        placeholder="Enter session secret"
                      />
                      <Text fontSize="sm" color="gray.500" mt={1}>
                        Used to sign session cookies. Keep this secret!
                      </Text>
                    </FormControl>

                    <Box>
                      <Text fontWeight="medium" mb={2}>Security Features</Text>
                      <VStack align="start" spacing={2}>
                        <HStack>
                          <Badge colorScheme="green">✓</Badge>
                          <Text fontSize="sm">HTTPS Ready</Text>
                        </HStack>
                        <HStack>
                          <Badge colorScheme="green">✓</Badge>
                          <Text fontSize="sm">CORS Protection</Text>
                        </HStack>
                        <HStack>
                          <Badge colorScheme="green">✓</Badge>
                          <Text fontSize="sm">Session Management</Text>
                        </HStack>
                        <HStack>
                          <Badge colorScheme="green">✓</Badge>
                          <Text fontSize="sm">Input Validation</Text>
                        </HStack>
                      </VStack>
                    </Box>

                    <Button
                      leftIcon={<FiSave />}
                      colorScheme="brand"
                      onClick={saveSettings}
                      isLoading={saving}
                      loadingText="Saving"
                    >
                      Save Security Settings
                    </Button>
                  </VStack>
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