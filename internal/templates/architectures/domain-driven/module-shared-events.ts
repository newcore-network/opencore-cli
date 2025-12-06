export const {{.ModuleNamePascal}}Events = {
  // Client to Server
  REQUEST: '{{.ModuleName}}:request',
  
  // Server to Client
  UPDATE: '{{.ModuleName}}:update',
  RESPONSE: '{{.ModuleName}}:response',
  
  // Internal events
  CREATED: '{{.ModuleName}}:created',
  UPDATED: '{{.ModuleName}}:updated',
  DELETED: '{{.ModuleName}}:deleted',
} as const;

