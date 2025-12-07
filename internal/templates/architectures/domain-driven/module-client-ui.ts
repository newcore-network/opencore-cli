import { Client } from '@open-core/framework';

@Client.Controller()
export class {{.ModuleNamePascal }}UI {
  constructor(private NUI: Client.NuiBridge) { }

  @Client.KeyMapping("f5", "Open {{.ModuleNamePascal}} UI")
  show() {
    // Show NUI or UI logic
    console.log('{{.ModuleNamePascal}} UI shown');
  }

  @Client.NuiCallback('{{.ModuleNamePascal}}:close')
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

