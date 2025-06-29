import React, { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import {
  Box,
  VStack,
  HStack,
  Text,
  Select,
  useColorModeValue,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Badge
} from '@chakra-ui/react'
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell
} from 'recharts'
import { gradients } from '../theme/gradients'

const MotionBox = motion(Box)

// Generate mock data
const generateTimeSeriesData = (days = 7) => {
  const data = []
  const now = new Date()
  
  for (let i = days; i >= 0; i--) {
    const date = new Date(now.getTime() - i * 24 * 60 * 60 * 1000)
    data.push({
      name: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
      requests: Math.floor(Math.random() * 1000) + 100,
      users: Math.floor(Math.random() * 50) + 10,
      errors: Math.floor(Math.random() * 20),
      responseTime: Math.floor(Math.random() * 100) + 50
    })
  }
  
  return data
}

const generateCollectionData = () => [
  { name: 'Users', value: 450, color: '#8884d8' },
  { name: 'Products', value: 320, color: '#82ca9d' },
  { name: 'Orders', value: 180, color: '#ffc658' },
  { name: 'Reviews', value: 90, color: '#ff7c7c' },
  { name: 'Categories', value: 60, color: '#8dd1e1' }
]

const CustomTooltip = ({ active, payload, label }) => {
  const bgColor = useColorModeValue('white', 'gray.800')
  const borderColor = useColorModeValue('gray.200', 'gray.600')
  
  if (active && payload && payload.length) {
    return (
      <Box
        bg={bgColor}
        p={3}
        borderRadius="lg"
        boxShadow="lg"
        borderWidth="1px"
        borderColor={borderColor}
      >
        <Text fontSize="sm" fontWeight="medium" mb={2}>
          {label}
        </Text>
        {payload.map((entry, index) => (
          <HStack key={index} spacing={2}>
            <Box
              w={3}
              h={3}
              bg={entry.color}
              borderRadius="full"
            />
            <Text fontSize="sm">
              {entry.name}: {entry.value}
            </Text>
          </HStack>
        ))}
      </Box>
    )
  }
  return null
}

export const MetricsChart = () => {
  const [timeRange, setTimeRange] = useState('7d')
  const [data, setData] = useState([])
  const [collectionData, setCollectionData] = useState([])
  
  const bgColor = useColorModeValue('white', 'gray.800')
  const textColor = useColorModeValue('gray.600', 'gray.300')
  const gridColor = useColorModeValue('#f0f0f0', '#2d3748')
  
  useEffect(() => {
    const days = timeRange === '24h' ? 1 : timeRange === '7d' ? 7 : 30
    setData(generateTimeSeriesData(days))
    setCollectionData(generateCollectionData())
  }, [timeRange])
  
  return (
    <MotionBox
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.6 }}
    >
      <VStack spacing={6} align="stretch">
        {/* Header */}
        <HStack justify="space-between" align="center">
          <VStack align="start" spacing={1}>
            <Text fontSize="xl" fontWeight="bold">
              Analytics Overview
            </Text>
            <Text fontSize="sm" color={textColor}>
              Real-time system metrics and usage statistics
            </Text>
          </VStack>
          
          <Select
            value={timeRange}
            onChange={(e) => setTimeRange(e.target.value)}
            size="sm"
            w="120px"
          >
            <option value="24h">Last 24h</option>
            <option value="7d">Last 7 days</option>
            <option value="30d">Last 30 days</option>
          </Select>
        </HStack>
        
        {/* Charts */}
        <Tabs variant="soft-rounded" colorScheme="brand">
          <TabList>
            <Tab>API Requests</Tab>
            <Tab>Response Time</Tab>
            <Tab>Collections</Tab>
            <Tab>Errors</Tab>
          </TabList>
          
          <TabPanels>
            {/* API Requests */}
            <TabPanel p={0} pt={6}>
              <Box h="300px">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={data}>
                    <defs>
                      <linearGradient id="requestsGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#667eea" stopOpacity={0.8}/>
                        <stop offset="95%" stopColor="#667eea" stopOpacity={0.1}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
                    <XAxis 
                      dataKey="name" 
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 12, fill: textColor }}
                    />
                    <YAxis 
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 12, fill: textColor }}
                    />
                    <Tooltip content={<CustomTooltip />} />
                    <Area
                      type="monotone"
                      dataKey="requests"
                      stroke="#667eea"
                      strokeWidth={3}
                      fill="url(#requestsGradient)"
                      dot={{ fill: '#667eea', strokeWidth: 2, r: 4 }}
                      activeDot={{ r: 6, fill: '#667eea' }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </Box>
            </TabPanel>
            
            {/* Response Time */}
            <TabPanel p={0} pt={6}>
              <Box h="300px">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={data}>
                    <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
                    <XAxis 
                      dataKey="name" 
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 12, fill: textColor }}
                    />
                    <YAxis 
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 12, fill: textColor }}
                    />
                    <Tooltip content={<CustomTooltip />} />
                    <Line
                      type="monotone"
                      dataKey="responseTime"
                      stroke="#4facfe"
                      strokeWidth={3}
                      dot={{ fill: '#4facfe', strokeWidth: 2, r: 4 }}
                      activeDot={{ r: 6, fill: '#4facfe' }}
                    />
                  </LineChart>
                </ResponsiveContainer>
              </Box>
            </TabPanel>
            
            {/* Collections */}
            <TabPanel p={0} pt={6}>
              <Box h="300px">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={collectionData}
                      cx="50%"
                      cy="50%"
                      innerRadius={60}
                      outerRadius={120}
                      paddingAngle={5}
                      dataKey="value"
                    >
                      {collectionData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </Pie>
                    <Tooltip content={<CustomTooltip />} />
                  </PieChart>
                </ResponsiveContainer>
              </Box>
            </TabPanel>
            
            {/* Errors */}
            <TabPanel p={0} pt={6}>
              <Box h="300px">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={data}>
                    <CartesianGrid strokeDasharray="3 3" stroke={gridColor} />
                    <XAxis 
                      dataKey="name" 
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 12, fill: textColor }}
                    />
                    <YAxis 
                      axisLine={false}
                      tickLine={false}
                      tick={{ fontSize: 12, fill: textColor }}
                    />
                    <Tooltip content={<CustomTooltip />} />
                    <Bar 
                      dataKey="errors" 
                      fill="#ff6b6b" 
                      radius={[4, 4, 0, 0]}
                    />
                  </BarChart>
                </ResponsiveContainer>
              </Box>
            </TabPanel>
          </TabPanels>
        </Tabs>
      </VStack>
    </MotionBox>
  )
}

export default MetricsChart