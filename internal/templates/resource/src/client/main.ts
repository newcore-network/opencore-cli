import { loggers } from '@open-core/framework'
import { Client } from '@open-core/framework/client'

// Bootstrap the resource client
Client.init({ mode: 'RESOURCE' })
  .catch((e: unknown) => loggers.bootstrap.error('Error found', { error: e }))
  .then(() => loggers.bootstrap.info('Client initialized!'))
