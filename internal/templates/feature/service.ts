import { Server } from '@open-core/framework/server';

@Server.Service()
export class {{.FeatureNamePascal}}Service {
  // Service logic here
  
  constructor() {
    console.log('{{.FeatureNamePascal}}Service initialized');
  }
}

