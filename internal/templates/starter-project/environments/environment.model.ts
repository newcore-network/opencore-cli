// Define here the shape of your environment configuration.
// Add any feature flags or settings that vary between environments.
export interface AppEnvironment {
  name: string
  production: boolean
  features: {
    debugLogs: boolean
    mockAccounts: boolean
  }
}
