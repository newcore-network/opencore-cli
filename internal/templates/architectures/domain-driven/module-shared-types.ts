export interface {{.ModuleNamePascal}}Data {
  id: number;
  playerId: number;
  // Add your data structure here
}

export interface {{.ModuleNamePascal}}Request {
  action: string;
  payload: any;
}

export interface {{.ModuleNamePascal}}Response {
  success: boolean;
  data?: any;
  error?: string;
}

