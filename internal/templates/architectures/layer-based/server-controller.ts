import { Server } from '@open-core/framework/server';
import { {{.FeatureNamePascal}}Service } from '../services/{{.FeatureName}}.service';

@Server.Controller()
export class {{.FeatureNamePascal}}Controller {
  constructor(private readonly {{.FeatureName}}Service: {{.FeatureNamePascal}}Service) {}

  @Server.Command('{{.FeatureName}}')
  async handleCommand(player: Server.Player, args: string[]) {
    const result = await this.{{.FeatureName}}Service.execute(player, args);
    console.log('Result:', result);
  }

  @Server.OnNet('{{.FeatureName}}:request')
  async handleRequest(player: Server.Player, data: any) {
    const response = await this.{{.FeatureName}}Service.process(player, data);
    emitNet('{{.FeatureName}}:response', player.source, response);
  }
}

