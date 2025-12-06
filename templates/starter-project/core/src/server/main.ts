import { Server } from '@open-core/framework';
{{if .InstallIdentity}}import '@open-core/identity';{{end}}

// Bootstrap the server
Server.bootstrap({
  features: [
    // Add your features here
  ],
});

console.log('{{.ProjectName}} server started!');

