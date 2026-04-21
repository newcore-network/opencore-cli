import type { AppEnvironment } from './environment.model'

export const environment: AppEnvironment = {
  name: 'development',
  production: false,
  features: {
    debugLogs: true,
    mockAccounts: true,
  },
}
