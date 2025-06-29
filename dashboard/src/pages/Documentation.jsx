import React, { useEffect } from 'react'
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
  Link,
  useColorModeValue,
  Icon,
} from '@chakra-ui/react'
import {
  FiBook,
  FiKey,
  FiUsers,
  FiDatabase,
  FiShield,
  FiCode,
  FiServer,
  FiExternalLink,
} from 'react-icons/fi'
import { AnimatedBackground } from '../components/AnimatedBackground'

function Documentation() {
  const cardBg = useColorModeValue('white', 'gray.700')

  useEffect(() => {
    // Redirect to GitHub docs after 3 seconds if user doesn't click
    const timer = setTimeout(() => {
      window.open('https://github.com/hjanuschka/go-deployd/tree/main/docs', '_blank')
    }, 5000)

    return () => clearTimeout(timer)
  }, [])

  const docLinks = [
    {
      title: 'Collections API',
      description: 'RESTful API for data operations, CRUD, and queries',
      icon: FiDatabase,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/collections-api.md'
    },
    {
      title: 'Authentication',
      description: 'JWT-based authentication and user management',
      icon: FiKey,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/authentication.md'
    },
    {
      title: 'Admin API',
      description: 'Server administration and management endpoints',
      icon: FiShield,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/admin-api.md'
    },
    {
      title: 'Events System',
      description: 'Server-side business logic in JavaScript or Go',
      icon: FiCode,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/events-system.md'
    },
    {
      title: 'Database Configuration',
      description: 'MongoDB, MySQL, and SQLite setup and configuration',
      icon: FiDatabase,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/database-config.md'
    },
    {
      title: 'WebSocket & Real-time',
      description: 'Real-time event broadcasting and WebSocket connections',
      icon: FiServer,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/websocket-realtime.md'
    },
    {
      title: 'dpd.js Client',
      description: 'JavaScript client library for browser and Node.js',
      icon: FiCode,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/dpd-js-client.md'
    },
    {
      title: 'Advanced Queries',
      description: 'MongoDB-style queries and SQL translation',
      icon: FiDatabase,
      url: 'https://github.com/hjanuschka/go-deployd/blob/main/docs/advanced-queries.md'
    }
  ]

  return (
    <Box minH="100vh" position="relative">
      <AnimatedBackground />
      
      <Box position="relative" zIndex={1} p={8}>
        <VStack spacing={8} align="stretch" maxW="1200px" mx="auto">
          <Card bg={cardBg} shadow="xl">
            <CardHeader textAlign="center">
              <VStack spacing={4}>
                <Icon as={FiBook} boxSize={12} color="blue.500" />
                <Heading size="xl">Go-Deployd Documentation</Heading>
                <Text fontSize="lg" color="gray.600">
                  Comprehensive documentation has been moved to GitHub for better version control and community contributions.
                </Text>
              </VStack>
            </CardHeader>
            <CardBody>
              <VStack spacing={6}>
                <HStack spacing={4} justify="center">
                  <Button
                    as={Link}
                    href="https://github.com/hjanuschka/go-deployd/blob/main/docs/index.md"
                    isExternal
                    colorScheme="blue"
                    size="lg"
                    rightIcon={<FiExternalLink />}
                  >
                    View Full Documentation
                  </Button>
                  <Button
                    as={Link}
                    href="https://github.com/hjanuschka/go-deployd/tree/main/docs"
                    isExternal
                    variant="outline"
                    size="lg"
                    rightIcon={<FiExternalLink />}
                  >
                    Browse All Docs
                  </Button>
                </HStack>

                <Text textAlign="center" fontSize="sm" color="gray.500">
                  You will be automatically redirected to the documentation in a few seconds...
                </Text>
              </VStack>
            </CardBody>
          </Card>

          <VStack spacing={4} align="stretch">
            <Heading size="lg" textAlign="center">Quick Links</Heading>
            <Box display="grid" gridTemplateColumns="repeat(auto-fit, minmax(300px, 1fr))" gap={4}>
              {docLinks.map((doc, index) => (
                <Card
                  key={index}
                  bg={cardBg}
                  shadow="md"
                  transition="all 0.2s"
                  _hover={{ shadow: "lg", transform: "translateY(-2px)" }}
                  cursor="pointer"
                  onClick={() => window.open(doc.url, '_blank')}
                >
                  <CardBody>
                    <VStack spacing={3} align="start">
                      <HStack>
                        <Icon as={doc.icon} color="blue.500" />
                        <Heading size="md">{doc.title}</Heading>
                        <Icon as={FiExternalLink} boxSize={3} color="gray.400" ml="auto" />
                      </HStack>
                      <Text fontSize="sm" color="gray.600">
                        {doc.description}
                      </Text>
                    </VStack>
                  </CardBody>
                </Card>
              ))}
            </Box>
          </VStack>

          <Card bg={cardBg} shadow="md">
            <CardBody>
              <VStack spacing={4} textAlign="center">
                <Heading size="md">Why GitHub Docs?</Heading>
                <HStack spacing={8} justify="center" wrap="wrap">
                  <VStack>
                    <Icon as={FiUsers} color="green.500" boxSize={6} />
                    <Text fontSize="sm">Community Contributions</Text>
                  </VStack>
                  <VStack>
                    <Icon as={FiCode} color="blue.500" boxSize={6} />
                    <Text fontSize="sm">Version Control</Text>
                  </VStack>
                  <VStack>
                    <Icon as={FiBook} color="purple.500" boxSize={6} />
                    <Text fontSize="sm">Searchable</Text>
                  </VStack>
                  <VStack>
                    <Icon as={FiExternalLink} color="orange.500" boxSize={6} />
                    <Text fontSize="sm">Always Up-to-Date</Text>
                  </VStack>
                </HStack>
              </VStack>
            </CardBody>
          </Card>
        </VStack>
      </Box>
    </Box>
  )
}

export default Documentation