import { Server } from '@open-core/framework';

@Server.Injectable()
export class {{.FeatureNamePascal}}Service {
  constructor() {
    console.log('{{.FeatureNamePascal}}Service initialized');
  }

  async execute(player: Server.Player, args: string[]) {
    // Business logic here
    console.log(`Executing for player ${player.source}`);
    return { success: true };
  }

  async process(player: Server.Player, data: any) {
    // Process data
    return { success: true, data };
  }
}

