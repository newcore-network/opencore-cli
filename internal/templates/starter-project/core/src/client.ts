// OpenCore Framework - Client Entry Point
import { loggers } from '@open-core/framework'
import { Client } from '@open-core/framework/client'

// OpenCore scans decorators automatically when is Marked as Controller(), if not, import here
// Example: import './my-feature.client';

Client.init({ mode: 'CORE' })
  .catch((e: unknown) => loggers.bootstrap.error('Error found', { error: e }))
  .then(() => loggers.bootstrap.info('Client initialized!'))
