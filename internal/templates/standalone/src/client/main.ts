import { loggers } from '@open-core/framework'
import { Client } from '@open-core/framework/client'

// Bootstrap the standalone client
Client.init({ mode: 'STANDALONE' })
  .catch((e: unknown) => loggers.bootstrap.error('Error found', { error: e }))
  .then(() => loggers.bootstrap.info('Client initialized!'))
