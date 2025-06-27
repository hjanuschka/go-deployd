export const gradients = {
  brand: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  success: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
  warning: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
  error: 'linear-gradient(135deg, #ff6b6b 0%, #ffa500 100%)',
  info: 'linear-gradient(135deg, #74b9ff 0%, #0984e3 100%)',
  purple: 'linear-gradient(135deg, #a8edea 0%, #fed6e3 100%)',
  blue: 'linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%)',
  green: 'linear-gradient(135deg, #ff9a9e 0%, #fecfef 100%)',
  dark: 'linear-gradient(135deg, #434343 0%, #000000 100%)',
  light: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
  ocean: 'linear-gradient(135deg, #2196f3 0%, #21cbf3 100%)',
  sunset: 'linear-gradient(135deg, #ffeee4 0%, #ffc3a0 100%)',
  aurora: 'linear-gradient(135deg, #667db6 0%, #0082c8 40%, #0082c8 60%, #667db6 100%)',
  cosmic: 'linear-gradient(135deg, #243949 0%, #517fa4 100%)',
  neon: 'linear-gradient(135deg, #ee0979 0%, #ff6a00 100%)',
  mint: 'linear-gradient(135deg, #56ab2f 0%, #a8e6cf 100%)'
}

export const getRandomGradient = () => {
  const keys = Object.keys(gradients)
  const randomKey = keys[Math.floor(Math.random() * keys.length)]
  return gradients[randomKey]
}