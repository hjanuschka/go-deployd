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
  Button,
  Select,
  FormControl,
  FormLabel,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Badge,
  Code,
  useColorModeValue,
  useToast,
  IconButton,
  Tooltip,
  Spinner,
  Center,
} from '@chakra-ui/react'
import {
  FiFileText,
  FiRefreshCw,
  FiDownload,
  FiFilter,
  FiClock,
  FiInfo,
  FiAlertTriangle,
  FiXCircle,
  FiUser,
} from 'react-icons/fi'
import { useAuth } from '../contexts/AuthContext'
import { AnimatedBackground } from '../components/AnimatedBackground'

function Logs() {
  const [logs, setLogs] = useState([])
  const [loading, setLoading] = useState(false)
  const [logLevel, setLogLevel] = useState('all')
  const [logFile, setLogFile] = useState('current')
  const [logFiles, setLogFiles] = useState([])
  
  const { authFetch } = useAuth()
  const toast = useToast()
  const codeBg = useColorModeValue('gray.50', 'gray.800')

  useEffect(() => {
    loadLogFiles()
    loadLogs()
  }, [logFile, logLevel])

  const loadLogFiles = async () => {
    try {
      const response = await authFetch('/_admin/logs/files')
      if (response.ok) {
        const data = await response.json()
        setLogFiles(data.files || [])
      }
    } catch (err) {
      console.error('Failed to load log files:', err)
    }
  }

  const loadLogs = async () => {
    try {
      setLoading(true)
      const params = new URLSearchParams()
      if (logLevel !== 'all') params.append('level', logLevel)
      if (logFile !== 'current') params.append('file', logFile)
      
      const response = await authFetch(`/_admin/logs?${params.toString()}`)
      if (response.ok) {
        const data = await response.json()
        setLogs(data.logs || [])
      } else {
        toast({
          title: 'Failed to load logs',
          status: 'error',
          duration: 3000,
          isClosable: true,
        })
      }
    } catch (err) {
      console.error('Failed to load logs:', err)
      toast({
        title: 'Error loading logs',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    } finally {
      setLoading(false)
    }
  }

  const downloadLogs = async () => {
    try {
      const params = new URLSearchParams()
      if (logLevel !== 'all') params.append('level', logLevel)
      if (logFile !== 'current') params.append('file', logFile)
      
      const response = await authFetch(`/_admin/logs/download?${params.toString()}`)
      if (response.ok) {
        const blob = await response.blob()
        const url = window.URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.style.display = 'none'
        a.href = url
        a.download = `deployd-logs-${logFile}-${new Date().toISOString().split('T')[0]}.jsonl`
        document.body.appendChild(a)
        a.click()
        window.URL.revokeObjectURL(url)
        document.body.removeChild(a)
        
        toast({
          title: 'Logs downloaded',
          status: 'success',
          duration: 2000,
          isClosable: true,
        })
      }
    } catch (err) {
      console.error('Failed to download logs:', err)
      toast({
        title: 'Download failed',
        description: err.message,
        status: 'error',
        duration: 3000,
        isClosable: true,
      })
    }
  }

  const getLevelIcon = (level) => {
    switch (level) {
      case 'info': return <FiInfo color="blue" />
      case 'warn': case 'warning': return <FiAlertTriangle color="orange" />
      case 'error': return <FiXCircle color="red" />
      case 'debug': return <FiFilter color="gray" />
      case 'user-generated': return <FiUser color="purple" />
      default: return <FiFileText />
    }
  }

  const getLevelColor = (level) => {
    switch (level) {
      case 'info': return 'blue'
      case 'warn': case 'warning': return 'orange'
      case 'error': return 'red'
      case 'debug': return 'gray'
      case 'user-generated': return 'purple'
      default: return 'gray'
    }
  }

  const formatTimestamp = (timestamp) => {
    try {
      return new Date(timestamp).toLocaleString()
    } catch {
      return timestamp
    }
  }

  return (
    <Box position="relative" minH="100vh">
      <AnimatedBackground />
      <Box position="relative" zIndex={1} p={6}>
        <VStack align="stretch" spacing={6}>
      <HStack justify="space-between">
        <HStack>
          <FiFileText />
          <Heading 
            size="lg" 
            color={useColorModeValue('gray.800', 'white')}
            bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
            px={4}
            py={2}
            borderRadius="lg"
            backdropFilter="blur(10px)"
          >
            Application Logs
          </Heading>
        </HStack>
        <HStack spacing={2}>
          <Tooltip label="Refresh logs">
            <IconButton
              icon={<FiRefreshCw />}
              onClick={loadLogs}
              variant="outline"
              size="sm"
              isLoading={loading}
            />
          </Tooltip>
          <Tooltip label="Download logs">
            <IconButton
              icon={<FiDownload />}
              onClick={downloadLogs}
              variant="outline"
              size="sm"
            />
          </Tooltip>
        </HStack>
      </HStack>


      <Box
        bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
        borderRadius="xl"
        p={6}
        backdropFilter="blur(20px)"
        borderWidth="1px"
        borderColor={useColorModeValue('gray.200', 'whiteAlpha.200')}
        boxShadow="xl"
      >
        <VStack align="stretch" spacing={4}>
          <HStack justify="space-between">
            <Heading 
              size="md"
              color={useColorModeValue('gray.800', 'white')}
            >
              Filter Logs
            </Heading>
            <Badge colorScheme="blue">{logs.length} entries</Badge>
          </HStack>
          <HStack spacing={4}>
            <FormControl maxW="200px">
              <FormLabel>Log Level</FormLabel>
              <Select value={logLevel} onChange={(e) => setLogLevel(e.target.value)}>
                <option value="all">All Levels</option>
                <option value="debug">Debug</option>
                <option value="info">Info</option>
                <option value="warn">Warning</option>
                <option value="error">Error</option>
                <option value="user-generated">User Generated</option>
              </Select>
            </FormControl>
            <FormControl maxW="200px">
              <FormLabel>Log File</FormLabel>
              <Select value={logFile} onChange={(e) => setLogFile(e.target.value)}>
                <option value="current">Current</option>
                {logFiles.map((file) => (
                  <option key={file} value={file}>
                    {file}
                  </option>
                ))}
              </Select>
            </FormControl>
          </HStack>
        </VStack>
      </Box>

      <Box
        bg={useColorModeValue('whiteAlpha.900', 'blackAlpha.600')}
        borderRadius="xl"
        p={6}
        backdropFilter="blur(20px)"
        borderWidth="1px"
        borderColor={useColorModeValue('gray.200', 'whiteAlpha.200')}
        boxShadow="xl"
      >
        <VStack align="stretch" spacing={4}>
          <HStack>
            <FiClock />
            <Heading 
              size="md"
              color={useColorModeValue('gray.800', 'white')}
            >
              Log Entries
            </Heading>
          </HStack>
          {loading ? (
            <Center py={8}>
              <VStack>
                <Spinner size="lg" color="brand.500" />
                <Text>Loading logs...</Text>
              </VStack>
            </Center>
          ) : logs.length === 0 ? (
            <Center py={8}>
              <VStack>
                <FiFileText size="48" color="gray" />
                <Text color="gray.500">No log entries found</Text>
                <Text fontSize="sm" color="gray.400">
                  Try adjusting the filters or check if logging is enabled
                </Text>
              </VStack>
            </Center>
          ) : (
            <VStack align="stretch" spacing={3}>
              {logs.map((log, index) => (
                <Box
                  key={index}
                  p={3}
                  borderRadius="md"
                  border="1px"
                  borderColor={useColorModeValue('gray.200', 'gray.600')}
                  bg={codeBg}
                >
                  <HStack justify="space-between" mb={2}>
                    <HStack>
                      {getLevelIcon(log.level)}
                      <Badge colorScheme={getLevelColor(log.level)} variant="subtle">
                        {log.level?.toUpperCase() || 'LOG'}
                      </Badge>
                      {log.source && (
                        <Badge variant="outline" fontSize="xs">
                          {log.source}
                        </Badge>
                      )}
                    </HStack>
                    <Text fontSize="xs" color="gray.500">
                      {formatTimestamp(log.timestamp)}
                    </Text>
                  </HStack>
                  <Text fontFamily="mono" fontSize="sm">
                    {log.message}
                  </Text>
                  {log.data && Object.keys(log.data).length > 0 && (
                    <Box mt={2}>
                      <Text fontSize="xs" color="gray.500" mb={1}>
                        Additional Data:
                      </Text>
                      <Box fontSize="xs" bg={useColorModeValue('gray.100', 'gray.900')} p={2} borderRadius="sm" fontFamily="mono" whiteSpace="pre">
                        <Code>{JSON.stringify(log.data, null, 2)}</Code>
                      </Box>
                    </Box>
                  )}
                </Box>
              ))}
            </VStack>
          )}
        </VStack>
      </Box>
        </VStack>
      </Box>
    </Box>
  )
}

export default Logs