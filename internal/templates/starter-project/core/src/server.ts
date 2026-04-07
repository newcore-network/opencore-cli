// OpenCore Framework - Server Entry Point
import { loggers } from '@open-core/framework'
import { Server } from '@open-core/framework/server'
{{if .InstallIdentity}}import '@open-core/identity'{{end}}

// OpenCore scans decorators automatically when is Marked as Controller(), if not, import here
// Example: import './my-feature.client';

Server.init({ mode: 'CORE'})
  .catch((e: unknown) => loggers.bootstrap.error('Error found', { error: e }))
  .then(() => loggers.bootstrap.info('Server initialized!'))
