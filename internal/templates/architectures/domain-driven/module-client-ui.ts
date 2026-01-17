import { Client } from '@open-core/framework/client';

@Client.Controller()
export class {{.ModuleNamePascal }}UI {
  constructor(private readonly NUI: Client.NuiBridge) { }

  @Client.Key("f5", "Open {{.ModuleNamePascal}} UI")
  show() {
    // Show NUI or UI logic
    console.log('{{.ModuleNamePascal}} UI shown');
  }

  @Client.OnView('{{.ModuleNamePascal}}:close')
  hide() {
    // Hide NUI or UI logic
    console.log('{{.ModuleNamePascal}} UI hidden');
  }

  @Client.OnNet('{{.ModuleNamePascal}}:update')
  update(data: any) {
    this.NUI.send('{{.ModuleNamePascal}}:update', data);
    console.log('UI updated with:', data);
  }
}

