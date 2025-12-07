import { Server, Player } from '@open-core/framework';
import { {{.FeatureNamePascal}}Service } from '../services/{{.FeatureName}}.service';

@Server.Controller()
export class {{.FeatureNamePascal}}Controller {
  constructor(private readonly {{.FeatureName}}Service: {{.FeatureNamePascal}}Service) {}

  @Server.Command('{{.FeatureName}}')
  async handleCommand(source: number, args: string[]) {
    const player = Player.fromSource(source);
    if (!player) return;

    const result = await this.{{.FeatureName}}Service.execute(player, args);
    console.log('Result:', result);
  }

  @Server.OnNet('{{.FeatureName}}:request')
  async handleRequest(source: number, data: any) {
    const player = Player.fromSource(source);
    if (!player) return;

    const response = await this.{{.FeatureName}}Service.process(player, data);
    emitNet('{{.FeatureName}}:response', source, response);
  }
}

