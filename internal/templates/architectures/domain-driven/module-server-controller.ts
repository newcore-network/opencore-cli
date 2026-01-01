import { Infer, Server, z } from '@open-core/framework';
import { {{.ModuleNamePascal}}Service } from './{{.ModuleName}}.service';

const schema = z.object({
  some: z.string().min(2).max(20)
})

@Server.Controller()
export class {{.ModuleNamePascal}}Controller {
  constructor(private readonly {{.ModuleName}}Service: {{.ModuleNamePascal}}Service) {}

  @Server.Command('{{.ModuleName}}')
  async handleCommand(player: Server.Player, args: string[]) {
    const result = await this.{{.ModuleName}}Service.execute(player, args);
    console.log('Command result:', result);
  }

  @Server.OnNet('{{.ModuleName}}:request', schema)
  async handleRequest(player: Server.Player, data: Infer<typeof schema>) {
    const response = await this.{{.ModuleName}}Service.processRequest(player, data);
    emitNet('{{.ModuleName}}:response', source, response);
  }
}

