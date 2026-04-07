import { loggers } from '@open-core/framework'
import { Server } from '@open-core/framework/server'

// Bootstrap the resource server
Server.init({ mode: 'RESOURCE', coreResourceName: 'core' })
  .catch((e: unknown) => loggers.bootstrap.error('Error found', { error: e }))
  .then(() => loggers.bootstrap.info('Server initialized!'))
