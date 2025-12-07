// OpenCore Framework - Server Entry Point
import { Server } from '@open-core/framework';
{{if .InstallIdentity}}import '@open-core/identity';{{end}}

// Register your controllers - OpenCore scans decorators automatically
// Example imports based on your architecture:
// import './modules/banking/server/banking.controller';
// import './features/jobs';

Server.init();

console.log('{{.ProjectName}} server initialized!');

