import { Server, Player } from '@open-core/framework/server';

@Server.Service()
export class {{.FeatureNamePascal}}Service {
  constructor() {
    console.log('{{.FeatureNamePascal}}Service initialized');
  }

  async execute(player: Player, args: string[]) {
    // Business logic here
    console.log(`Executing for player ${player.source}`);
    return { success: true };
  }

  async process(player: Player, data: any) {
    // Process data
    return { success: true, data };
  }
}

