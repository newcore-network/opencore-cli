import { Server } from '@open-core/framework';

@Server.Injectable()
export class {{.FeatureNamePascal}}Service {
  // Service logic here
  
  constructor() {
    console.log('{{.FeatureNamePascal}}Service initialized');
  }
}

