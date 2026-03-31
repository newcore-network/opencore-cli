import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const { createOpenCoreViteConfig } = require('./vite.js')

export { createOpenCoreViteConfig }
export default createOpenCoreViteConfig
