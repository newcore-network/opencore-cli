import { Client } from '@open-core/framework';
import { {{.ModuleNamePascal}}UI } from './{{.ModuleName}}.ui';

@Client.Controller()
export class {{.ModuleNamePascal}}Controller {
  constructor(private readonly ui: {{.ModuleNamePascal}}UI) {}

  @Client.OnNet('{{.ModuleName}}:update')
  handleUpdate(data: any) {
    console.log('Received update:', data);
    this.ui.update(data);
  }

  @Client.Key('F5', 'Open {{.ModuleNamePascal}}')
  openUI() {
    this.ui.show();
  }
}

