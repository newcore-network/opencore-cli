// OpenCore Framework - Client Entry Point
import { Client } from '@open-core/framework';

// Register your client controllers - OpenCore scans decorators automatically
// Example imports based on your architecture:
// import './modules/hud/client/hud.controller';
// import './features/interface';

Client.init();

console.log('{{.ProjectName}} client initialized!');

