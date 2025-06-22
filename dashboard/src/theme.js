import { extendTheme } from '@chakra-ui/react'

const theme = extendTheme({
  config: {
    initialColorMode: 'light',
    useSystemColorMode: false,
  },
  colors: {
    brand: {
      50: '#e3f2fd',
      100: '#bbdefb',
      200: '#90caf9',
      300: '#64b5f6',
      400: '#42a5f5',
      500: '#2196f3', // Primary blue
      600: '#1e88e5',
      700: '#1976d2',
      800: '#1565c0',
      900: '#0d47a1',
    },
    accent: {
      50: '#fbe9e7',
      100: '#ffccbc',
      200: '#ffab91',
      300: '#ff8a65',
      400: '#ff7043',
      500: '#ff5722', // Secondary orange
      600: '#f4511e',
      700: '#e64a19',
      800: '#d84315',
      900: '#bf360c',
    },
  },
  fonts: {
    heading: 'Roboto, Arial, sans-serif',
    body: 'Roboto, Arial, sans-serif',
  },
  styles: {
    global: (props) => ({
      body: {
        bg: props.colorMode === 'dark' ? 'gray.800' : 'gray.50',
        color: props.colorMode === 'dark' ? 'white' : 'gray.800',
      },
    }),
  },
  components: {
    Button: {
      defaultProps: {
        colorScheme: 'brand',
      },
    },
    Card: {
      baseStyle: {
        container: {
          boxShadow: 'md',
          borderRadius: 'lg',
          bg: 'white',
          _dark: {
            bg: 'gray.700',
          },
        },
      },
    },
    Drawer: {
      parts: ['dialog'],
      baseStyle: {
        dialog: {
          bg: 'gray.900',
          color: 'white',
        },
      },
    },
  },
})

export default theme