import { Client } from '@open-core/framework';

@Client.Controller()
export class {{.FeatureNamePascal}}Controller {
  constructor() {}

  @Client.OnNet('{{.FeatureName}}:update')
  handleUpdate(data: any) {
    console.log('Received update:', data);
  }

  @Client.KeyMapping('F5', 'Toggle {{.FeatureNamePascal}}')
  toggle() {
    console.log('{{.FeatureNamePascal}} toggled');
  }
}

