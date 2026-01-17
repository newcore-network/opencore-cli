import { Client } from '@open-core/framework/client';

@Client.Controller()
export class {{.FeatureNamePascal}}Controller {
  constructor() {}

  @Client.OnNet('{{.FeatureName}}:response')
  handleResponse(data: any) {
    console.log('Received response:', data);
  }
}
