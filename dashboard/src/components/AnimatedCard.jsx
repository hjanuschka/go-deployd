import React from 'react'
import { motion } from 'framer-motion'
import { Card } from '@chakra-ui/react'

const MotionCard = motion(Card)

export const AnimatedCard = ({ children, delay = 0, ...props }) => {
  return (
    <MotionCard
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5, delay }}
      whileHover={{ 
        y: -4,
        boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)'
      }}
      whileTap={{ scale: 0.98 }}
      cursor="pointer"
      {...props}
    >
      {children}
    </MotionCard>
  )
}

export default AnimatedCard