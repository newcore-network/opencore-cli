import { Client } from '@open-core/framework/client';

@Client.Service()
export class {{.FeatureNamePascal}}ClientService {
  private state: any = {};

  updateState(data: any) {
    this.state = { ...this.state, ...data };
    console.log('Client state updated:', this.state);
  }

  getState() {
    return this.state;
  }
}

