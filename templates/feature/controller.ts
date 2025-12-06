import { Controller, Command } from '@open-core/framework';
import { {{.FeatureNamePascal}}Service } from './{{.FeatureName}}.service';

@Controller()
export class {{.FeatureNamePascal}}Controller {
  constructor(private readonly {{.FeatureName}}Service: {{.FeatureNamePascal}}Service) {}

  @Command('{{.FeatureName}}')
  handle(source: number, args: string[]) {
    // Command logic here
    console.log('{{.FeatureName}} command executed');
  }
}

