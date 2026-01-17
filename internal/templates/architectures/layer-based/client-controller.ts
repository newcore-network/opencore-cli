import { Client } from '@open-core/framework/client';
import { {{.FeatureNamePascal}}ClientService } from '../services/{{.FeatureName}}.client.service';

@Client.Controller()
export class {{.FeatureNamePascal}}Controller {
  constructor(private readonly {{.FeatureName}}Service: {{.FeatureNamePascal}}ClientService) {}

  @Client.OnNet('{{.FeatureName}}:update')
  handleUpdate(data: any) {
    this.{{.FeatureName}}Service.updateState(data);
  }

  @Client.Key('F5', 'Toggle {{.FeatureNamePascal}}')
  toggle() {
    const state = this.{{.FeatureName}}Service.getState();
    console.log('{{.FeatureNamePascal}} state:', state);
  }
}

