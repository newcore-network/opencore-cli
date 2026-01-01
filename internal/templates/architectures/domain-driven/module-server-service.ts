import { Server, Player } from '@open-core/framework';
import { {{.ModuleNamePascal}}Repository } from './{{.ModuleName}}.repository';

@Server.Service()
export class {{.ModuleNamePascal}}Service {
  constructor(private readonly repository: {{.ModuleNamePascal}}Repository) {}

  async execute(player: Player, args: string[]) {
    // Business logic here
    console.log(`Executing {{.ModuleName}} for player ${player.source}`);
    
    const data = await this.repository.findByPlayer(player.source);
    return { success: true, data };
  }

  async processRequest(player: Player, data: any) {
    // Process request logic
    const result = await this.repository.save(player.source, data);
    return result;
  }
}

