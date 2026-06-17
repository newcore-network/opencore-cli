import type { AppEnvironment } from './environment.model'

export const environment: AppEnvironment = {
  name: 'production',
  production: true,
  features: {
    debugLogs: false,
    mockAccounts: false,
  },
}
