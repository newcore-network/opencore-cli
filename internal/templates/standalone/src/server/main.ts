// {{.StandaloneName}} - Server Side
import { loggers } from '@open-core/framework'
import { init } from '@open-core/framework/server'

// Bootstrap the standalone server
// Standalone mode enables guards and decorators without depending on a core resource
init({ mode: 'STANDALONE' })
  .catch((e: unknown) => loggers.bootstrap.error('Error found', { error: e }))
  .then(() => loggers.bootstrap.info('Server initialized!'))