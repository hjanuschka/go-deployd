import React, { useState, useEffect } from 'react';
import {
  Box,
  SimpleGrid,
  Card,
  CardBody,
  CardHeader,
  Heading,
  Text,
  VStack,
  HStack,
  Icon,
  Button,
  Alert,
  AlertIcon,
  AlertDescription,
  Spinner,
  Center,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  useColorModeValue,
  Flex,
  Badge,
} from '@chakra-ui/react';
import { 
  LineChart, 
  Line, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell
} from 'recharts';
import { FiActivity, FiDatabase, FiAlertTriangle, FiServer, FiClock } from 'react-icons/fi';

const COLORS = ['#3182CE', '#38A169', '#D69E2E', '#E53E3E', '#00B5D8'];

export default function Metrics() {
  const [systemStats, setSystemStats] = useState(null);
  const [detailedMetrics, setDetailedMetrics] = useState([]);
  const [aggregatedMetrics, setAggregatedMetrics] = useState([]);
  const [eventMetrics, setEventMetrics] = useState({});
  const [collections, setCollections] = useState(['all']);
  const [selectedCollection, setSelectedCollection] = useState('overall');
  const [timeRange, setTimeRange] = useState('24h');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  const cardBg = useColorModeValue('white', 'gray.800');
  const borderColor = useColorModeValue('gray.200', 'gray.700');

  useEffect(() => {
    fetchCollections();
    fetchMetrics();
    const interval = setInterval(fetchMetrics, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, [timeRange, selectedCollection]);

  const fetchCollections = async () => {
    try {
      const response = await fetch('/_dashboard/api/metrics/collections', {
        headers: {
          'X-Master-Key': localStorage.getItem('masterKey'),
        },
      });
      if (response.ok) {
        const data = await response.json();
        setCollections(data.collections || ['all']);
      }
    } catch (err) {
      console.warn('Failed to fetch collections:', err);
    }
  };

  const fetchMetrics = async () => {
    try {
      setLoading(true);
      setError(null);

      const headers = {
        'X-Master-Key': localStorage.getItem('masterKey'),
      };

      // Build collection query parameter
      const collectionParam = selectedCollection !== 'all' ? `?collection=${selectedCollection}` : '';

      // Fetch system stats
      const systemResponse = await fetch('/_dashboard/api/metrics/system', { headers });
      if (!systemResponse.ok) throw new Error('Failed to fetch system stats');
      const systemData = await systemResponse.json();
      setSystemStats(systemData);

      // Fetch detailed metrics (last 24h)
      const detailedResponse = await fetch(`/_dashboard/api/metrics/detailed${collectionParam}`, { headers });
      if (!detailedResponse.ok) throw new Error('Failed to fetch detailed metrics');
      const detailedData = await detailedResponse.json();
      setDetailedMetrics(detailedData.metrics || []);

      // Fetch aggregated metrics based on time range
      let aggregatedPeriod = 'hourly';
      if (timeRange === '6m') aggregatedPeriod = 'daily';
      if (timeRange === '12m') aggregatedPeriod = 'monthly';

      const aggregatedResponse = await fetch(`/_dashboard/api/metrics/periods?period=${aggregatedPeriod}${collectionParam}`, { headers });
      if (!aggregatedResponse.ok) throw new Error('Failed to fetch aggregated metrics');
      const aggregatedData = await aggregatedResponse.json();
      setAggregatedMetrics(aggregatedData.metrics || []);

      // Fetch event-specific metrics for selected collection
      if (selectedCollection !== 'all') {
        const eventResponse = await fetch(`/_dashboard/api/metrics/events?collection=${selectedCollection}&period=${aggregatedPeriod}`, { headers });
        if (eventResponse.ok) {
          const eventData = await eventResponse.json();
          setEventMetrics(eventData.events || {});
        }
      } else {
        setEventMetrics({});
      }

    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const formatDuration = (nanoseconds) => {
    const ms = nanoseconds / 1000000;
    if (ms < 1) return `${(nanoseconds / 1000).toFixed(1)}μs`;
    if (ms < 1000) return `${ms.toFixed(1)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  const formatUptime = (seconds) => {
    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${days}d ${hours}h ${minutes}m`;
  };

  // Process metrics for charts
  const processChartData = () => {
    if (!aggregatedMetrics.length) return [];
    
    return aggregatedMetrics
      .sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp))
      .map(metric => ({
        time: new Date(metric.timestamp).toLocaleDateString(),
        requests: metric.request_count,
        errors: metric.error_count,
        avgDuration: metric.avg_duration / 1000000, // Convert to ms
        errorRate: metric.error_rate
      }));
  };

  const getMetricsByType = () => {
    const types = { 'HTTP Requests': 0, 'Database Ops': 0, 'Hook Calls': 0, 'Errors': 0 };
    
    detailedMetrics.forEach(metric => {
      switch (metric.type) {
        case 0: types['HTTP Requests']++; break;
        case 1: types['Database Ops']++; break;
        case 2: types['Hook Calls']++; break;
        case 3: types['Errors']++; break;
      }
    });

    return Object.entries(types).map(([name, value]) => ({ name, value }));
  };

  const getRecentErrors = () => {
    return detailedMetrics
      .filter(metric => metric.error)
      .slice(-10)
      .reverse();
  };

  if (loading && !systemStats) {
    return (
      <Center h="400px">
        <VStack spacing={4}>
          <Spinner size="xl" color="brand.500" />
          <Text>Loading metrics...</Text>
        </VStack>
      </Center>
    );
  }

  if (error) {
    return (
      <Alert status="error">
        <AlertIcon />
        <AlertDescription>Error loading metrics: {error}</AlertDescription>
      </Alert>
    );
  }

  const chartData = processChartData();
  const metricsByType = getMetricsByType();
  const recentErrors = getRecentErrors();

  return (
    <VStack spacing={6} align="stretch">
      <Flex justify="space-between" align="center" wrap="wrap" gap={4}>
        <Heading size="lg">Performance Metrics</Heading>
        <HStack spacing={4}>
          {/* Collection Selector */}
          <Box minW="150px">
            <Text fontSize="sm" mb={1}>Collection:</Text>
            <select
              value={selectedCollection}
              onChange={(e) => setSelectedCollection(e.target.value)}
              style={{
                padding: '4px 8px',
                borderRadius: '4px',
                border: '1px solid #e2e8f0',
                fontSize: '14px',
                width: '100%'
              }}
            >
              {collections.map(collection => (
                <option key={collection} value={collection}>
                  {collection === 'all' ? 'All Collections' : collection}
                </option>
              ))}
            </select>
          </Box>

          {/* Time Range Selector */}
          <Box>
            <Text fontSize="sm" mb={1}>Time Range:</Text>
            <HStack>
              <Button
                size="sm"
                variant={timeRange === '24h' ? 'solid' : 'outline'}
                colorScheme="brand"
                onClick={() => setTimeRange('24h')}
              >
                24h
              </Button>
              <Button
                size="sm"
                variant={timeRange === '7d' ? 'solid' : 'outline'}
                colorScheme="brand"
                onClick={() => setTimeRange('7d')}
              >
                7d
              </Button>
              <Button
                size="sm"
                variant={timeRange === '6m' ? 'solid' : 'outline'}
                colorScheme="brand"
                onClick={() => setTimeRange('6m')}
              >
                6m
              </Button>
              <Button
                size="sm"
                variant={timeRange === '12m' ? 'solid' : 'outline'}
                colorScheme="brand"
                onClick={() => setTimeRange('12m')}
              >
                12m
              </Button>
            </HStack>
          </Box>
        </HStack>
      </Flex>

      {/* System Overview Cards */}
      <SimpleGrid columns={{ base: 1, md: 2, lg: 4 }} spacing={4}>
        <Card bg={cardBg} borderColor={borderColor}>
          <CardBody>
            <Stat>
              <Flex justify="space-between" align="start">
                <Box>
                  <StatLabel>Uptime</StatLabel>
                  <StatNumber fontSize="2xl">
                    {systemStats ? formatUptime(systemStats.uptime_seconds) : '-'}
                  </StatNumber>
                </Box>
                <Icon as={FiServer} boxSize={6} color="gray.400" />
              </Flex>
            </Stat>
          </CardBody>
        </Card>

        <Card bg={cardBg} borderColor={borderColor}>
          <CardBody>
            <Stat>
              <Flex justify="space-between" align="start">
                <Box>
                  <StatLabel>Requests/Hour</StatLabel>
                  <StatNumber fontSize="2xl">
                    {systemStats ? systemStats.hourly_requests.toLocaleString() : '-'}
                  </StatNumber>
                </Box>
                <Icon as={FiActivity} boxSize={6} color="gray.400" />
              </Flex>
            </Stat>
          </CardBody>
        </Card>

        <Card bg={cardBg} borderColor={borderColor}>
          <CardBody>
            <Stat>
              <Flex justify="space-between" align="start">
                <Box>
                  <StatLabel>Error Rate</StatLabel>
                  <StatNumber fontSize="2xl" color={systemStats?.hourly_error_rate > 5 ? 'red.500' : 'green.500'}>
                    {systemStats ? `${systemStats.hourly_error_rate.toFixed(1)}%` : '-'}
                  </StatNumber>
                </Box>
                <Icon as={FiAlertTriangle} boxSize={6} color="gray.400" />
              </Flex>
            </Stat>
          </CardBody>
        </Card>

        <Card bg={cardBg} borderColor={borderColor}>
          <CardBody>
            <Stat>
              <Flex justify="space-between" align="start">
                <Box>
                  <StatLabel>Total Metrics</StatLabel>
                  <StatNumber fontSize="2xl">
                    {systemStats ? systemStats.total_metrics.toLocaleString() : '-'}
                  </StatNumber>
                </Box>
                <Icon as={FiDatabase} boxSize={6} color="gray.400" />
              </Flex>
            </Stat>
          </CardBody>
        </Card>
      </SimpleGrid>

      {/* Charts Row */}
      <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={6}>
        {/* Request Volume Chart */}
        <Card bg={cardBg} borderColor={borderColor}>
          <CardHeader>
            <Heading size="md">Request Volume Over Time</Heading>
          </CardHeader>
          <CardBody>
            <Box h="300px">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <Tooltip />
                  <Line type="monotone" dataKey="requests" stroke="#3182CE" strokeWidth={2} />
                </LineChart>
              </ResponsiveContainer>
            </Box>
          </CardBody>
        </Card>

        {/* Response Times */}
        <Card bg={cardBg} borderColor={borderColor}>
          <CardHeader>
            <Heading size="md">Average Response Time</Heading>
          </CardHeader>
          <CardBody>
            <Box h="300px">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <Tooltip formatter={(value) => [`${value.toFixed(2)}ms`, 'Avg Duration']} />
                  <Line type="monotone" dataKey="avgDuration" stroke="#38A169" strokeWidth={2} />
                </LineChart>
              </ResponsiveContainer>
            </Box>
          </CardBody>
        </Card>

        {/* Metrics by Type */}
        <Card bg={cardBg} borderColor={borderColor}>
          <CardHeader>
            <Heading size="md">Metrics by Type (24h)</Heading>
          </CardHeader>
          <CardBody>
            <Box h="300px">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={metricsByType}
                    cx="50%"
                    cy="50%"
                    labelLine={false}
                    label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                    outerRadius={80}
                    fill="#8884d8"
                    dataKey="value"
                  >
                    {metricsByType.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip />
                </PieChart>
              </ResponsiveContainer>
            </Box>
          </CardBody>
        </Card>

        {/* Error Rate Chart */}
        <Card bg={cardBg} borderColor={borderColor}>
          <CardHeader>
            <Heading size="md">Error Rate Over Time</Heading>
          </CardHeader>
          <CardBody>
            <Box h="300px">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="time" />
                  <YAxis />
                  <Tooltip formatter={(value) => [`${value.toFixed(1)}%`, 'Error Rate']} />
                  <Bar dataKey="errorRate" fill="#E53E3E" />
                </BarChart>
              </ResponsiveContainer>
            </Box>
          </CardBody>
        </Card>
      </SimpleGrid>

      {/* Event Performance (when collection is selected) */}
      {selectedCollection !== 'all' && Object.keys(eventMetrics).length > 0 && (
        <Card bg={cardBg} borderColor={borderColor}>
          <CardHeader>
            <Heading size="md">Event Performance - {selectedCollection}</Heading>
          </CardHeader>
          <CardBody>
            <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={4}>
              {Object.entries(eventMetrics).map(([eventName, metrics]) => {
                const avgDuration = metrics.reduce((sum, m) => sum + (m.duration || 0), 0) / metrics.length / 1000000; // Convert to ms
                const errorCount = metrics.filter(m => m.error).length;
                const errorRate = (errorCount / metrics.length) * 100;

                return (
                  <Box key={eventName} p={4} borderWidth="1px" borderRadius="md" borderColor={borderColor}>
                    <VStack align="start" spacing={2}>
                      <Text fontWeight="bold" fontSize="sm">{eventName}</Text>
                      <HStack justify="space-between" w="full">
                        <Text fontSize="xs" color="gray.600">Avg Duration:</Text>
                        <Text fontSize="xs" fontWeight="medium">{avgDuration.toFixed(2)}ms</Text>
                      </HStack>
                      <HStack justify="space-between" w="full">
                        <Text fontSize="xs" color="gray.600">Calls:</Text>
                        <Text fontSize="xs" fontWeight="medium">{metrics.length}</Text>
                      </HStack>
                      <HStack justify="space-between" w="full">
                        <Text fontSize="xs" color="gray.600">Error Rate:</Text>
                        <Text fontSize="xs" fontWeight="medium" color={errorRate > 5 ? 'red.500' : 'green.500'}>
                          {errorRate.toFixed(1)}%
                        </Text>
                      </HStack>
                    </VStack>
                  </Box>
                );
              })}
            </SimpleGrid>
          </CardBody>
        </Card>
      )}

      {/* Recent Errors */}
      {recentErrors.length > 0 && (
        <Card bg={cardBg} borderColor={borderColor}>
          <CardHeader>
            <Heading size="md">Recent Errors</Heading>
          </CardHeader>
          <CardBody>
            <VStack spacing={3} align="stretch">
              {recentErrors.map((metric, index) => (
                <Box key={index} p={3} bg="red.50" borderRadius="md" borderLeft="4px" borderLeftColor="red.500">
                  <Flex justify="space-between" align="center">
                    <HStack spacing={2}>
                      <Icon as={FiAlertTriangle} color="red.500" />
                      <Text fontFamily="mono" fontSize="sm">
                        {metric.path || metric.metadata?.collection}
                      </Text>
                      <Badge colorScheme="red" variant="subtle">
                        {metric.error}
                      </Badge>
                    </HStack>
                    <VStack spacing={0} align="end">
                      <Text fontSize="sm" color="gray.600">
                        {formatDuration(metric.duration)}
                      </Text>
                      <Text fontSize="xs" color="gray.500">
                        {new Date(metric.timestamp).toLocaleTimeString()}
                      </Text>
                    </VStack>
                  </Flex>
                </Box>
              ))}
            </VStack>
          </CardBody>
        </Card>
      )}
    </VStack>
  );
}