// OpenCore Framework - Client Entry Point (Layer-Based)
import { Client } from '@open-core/framework';

// Register your client controllers - OpenCore scans decorators automatically
// import './controllers/hud.controller';
// import './services/ui.service';

Client.init();

console.log('{{.ProjectName}} client initialized!');

