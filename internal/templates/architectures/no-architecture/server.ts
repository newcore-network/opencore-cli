import { Server } from '@open-core/framework/server';

@Server.Controller()
export class {{.FeatureNamePascal}}Controller {
  constructor() {}

  @Server.Command('{{.FeatureName}}')
  async handleCommand(player: Server.Player, args: string[]) {
    console.log('Command executed by:', player.source);
  }
}
