// OpenCore Framework - Server Entry Point (Layer-Based)
import { Server } from '@open-core/framework/server';
{{if .InstallIdentity}}import '@open-core/identity';{{end}}

// Register your controllers - OpenCore scans decorators automatically
// import './controllers/banking.controller';
// import './services/banking.service';

Server.init();

console.log('{{.ProjectName}} server initialized!');

