import React from 'react'
import { Routes, Route } from 'react-router-dom'
import {
  Box,
  Flex,
  VStack,
  HStack,
  Text,
  Heading,
  IconButton,
  Drawer,
  DrawerBody,
  DrawerOverlay,
  DrawerContent,
  DrawerCloseButton,
  useDisclosure,
  useColorModeValue,
  useColorMode,
  Button,
  Spinner,
  Center,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
} from '@chakra-ui/react'
import {
  HamburgerIcon,
  MoonIcon,
  SunIcon,
  ChevronDownIcon,
} from '@chakra-ui/icons'
import {
  FiHome,
  FiDatabase,
  FiTool,
  FiSettings,
  FiCode,
  FiLogOut,
  FiKey,
  FiBook,
  FiFileText,
  FiActivity,
  FiUsers,
} from 'react-icons/fi'
import { useNavigate, useLocation } from 'react-router-dom'

import Dashboard from './pages/Dashboard'
import Collections from './pages/Collections'
import CollectionDetail from './pages/CollectionDetail'
import Users from './pages/Users'
import ApiTester from './pages/ApiTester'
import Documentation from './pages/Documentation'
import Logs from './pages/Logs'
import Settings from './pages/Settings'
import Login from './pages/Login'
import Metrics from './pages/Metrics'
import Sidebar from './components/Sidebar'
import { AuthProvider, useAuth } from './contexts/AuthContext'

const menuItems = [
  { text: 'Dashboard', icon: FiHome, path: '/' },
  { text: 'Collections', icon: FiDatabase, path: '/collections' },
  { text: 'Users', icon: FiUsers, path: '/users' },
  { text: 'API Tester', icon: FiTool, path: '/api-tester' },
  { text: 'Metrics', icon: FiActivity, path: '/metrics' },
  { text: 'Documentation', icon: FiBook, path: '/documentation' },
  { text: 'Logs', icon: FiFileText, path: '/logs' },
  { text: 'Settings', icon: FiSettings, path: '/settings' }
]

function AuthenticatedApp() {
  const { isOpen, onOpen, onClose } = useDisclosure()
  const { colorMode, toggleColorMode } = useColorMode()
  const { logout, masterKey } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()

  const bg = useColorModeValue('white', 'gray.800')
  const borderColor = useColorModeValue('gray.200', 'gray.700')

  const getCurrentPageTitle = () => {
    const currentItem = menuItems.find(item => item.path === location.pathname)
    return currentItem?.text || 'Dashboard'
  }

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <Flex minH="100vh" bg={useColorModeValue('gray.50', 'gray.900')}>
      {/* Desktop Sidebar */}
      <Box
        display={{ base: 'none', md: 'block' }}
        w="240px"
        bg="gray.900"
        color="white"
        position="fixed"
        h="full"
        overflowY="auto"
      >
        <Sidebar menuItems={menuItems} />
      </Box>

      {/* Mobile Drawer */}
      <Drawer isOpen={isOpen} placement="left" onClose={onClose}>
        <DrawerOverlay />
        <DrawerContent bg="gray.900" color="white">
          <DrawerCloseButton />
          <DrawerBody p={0}>
            <Sidebar menuItems={menuItems} onClose={onClose} />
          </DrawerBody>
        </DrawerContent>
      </Drawer>

      {/* Main Content */}
      <Box flex="1" ml={{ base: 0, md: '240px' }}>
        {/* Top Bar */}
        <Flex
          as="header"
          align="center"
          justify="space-between"
          w="full"
          px={4}
          py={4}
          bg={bg}
          borderBottomWidth="1px"
          borderColor={borderColor}
          position="sticky"
          top={0}
          zIndex={10}
        >
          <HStack spacing={4}>
            <IconButton
              display={{ base: 'flex', md: 'none' }}
              onClick={onOpen}
              variant="outline"
              aria-label="Open menu"
              icon={<HamburgerIcon />}
            />
            <Heading size="lg" color="brand.500">
              {getCurrentPageTitle()}
            </Heading>
          </HStack>

          <HStack spacing={2}>
            <Menu>
              <MenuButton as={Button} rightIcon={<ChevronDownIcon />} variant="ghost" size="sm">
                <HStack spacing={2}>
                  <FiKey />
                  <Text>Admin</Text>
                </HStack>
              </MenuButton>
              <MenuList>
                <MenuItem icon={<FiLogOut />} onClick={handleLogout}>
                  Logout
                </MenuItem>
              </MenuList>
            </Menu>
            <IconButton
              onClick={toggleColorMode}
              variant="ghost"
              aria-label="Toggle color mode"
              icon={colorMode === 'light' ? <MoonIcon /> : <SunIcon />}
            />
          </HStack>
        </Flex>

        {/* Page Content */}
        <Box p={6}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/collections" element={<Collections />} />
            <Route path="/collections/:name" element={<CollectionDetail />} />
            <Route path="/users" element={<Users />} />
            <Route path="/api-tester" element={<ApiTester />} />
            <Route path="/metrics" element={<Metrics />} />
            <Route path="/documentation" element={<Documentation />} />
            <Route path="/logs" element={<Logs />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </Box>
      </Box>
    </Flex>
  )
}

function App() {
  const { isAuthenticated, loading, login } = useAuth()

  if (loading) {
    return (
      <Center minH="100vh">
        <VStack spacing={4}>
          <Spinner size="xl" color="brand.500" />
          <Text>Loading...</Text>
        </VStack>
      </Center>
    )
  }

  if (!isAuthenticated) {
    return <Login onLogin={login} />
  }

  return <AuthenticatedApp />
}

function AppWithAuth() {
  return (
    <AuthProvider>
      <App />
    </AuthProvider>
  )
}

export default AppWithAuth