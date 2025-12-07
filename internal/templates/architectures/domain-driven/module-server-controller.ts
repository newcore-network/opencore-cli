import { Server } from '@open-core/framework';
import { {{.ModuleNamePascal}}Service } from './{{.ModuleName}}.service';

@Server.Controller()
export class {{.ModuleNamePascal}}Controller {
  constructor(private readonly {{.ModuleName}}Service: {{.ModuleNamePascal}}Service) {}

  @Server.Command('{{.ModuleName}}')
  async handleCommand(source: number, args: string[]) {
    const player = Server.Player.fromSource(source);
    if (!player) return;

    const result = await this.{{.ModuleName}}Service.execute(player, args);
    console.log('Command result:', result);
  }

  @Server.OnNet('{{.ModuleName}}:request')
  async handleRequest(source: number, data: any) {
    const player = Server.Player.fromSource(source);
    if (!player) return;

    const response = await this.{{.ModuleName}}Service.processRequest(player, data);
    emitNet('{{.ModuleName}}:response', source, response);
  }
}

