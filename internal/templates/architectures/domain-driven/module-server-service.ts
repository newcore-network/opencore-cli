import { Server, Player } from '@open-core/framework';
import { {{.ModuleNamePascal}}Repository } from './{{.ModuleName}}.repository';
import { schema } from './{{.ModuleName}}.controller';

@Server.Service()
export class {{.ModuleNamePascal}}Service {
  constructor(private readonly repository: {{.ModuleNamePascal}}Repository) {}

  async execute(player: Server.Player, args: string[]) {
    // Business logic here
    console.log(`Executing {{.ModuleName}} for player ${player.clientID}`);
    
    const data = await this.repository.findByPlayer(player.clientID);
    return { success: true, data };
  }

  async processRequest(player: Server.Player, data: Infer<typeof schema>) {
    // Process request logic
    const result = await this.repository.save(player.clientID, data);
    return result;
  }
}

