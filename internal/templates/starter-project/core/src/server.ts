// OpenCore Framework - Server Entry Point
import { Server } from '@open-core/framework/server';
{{if .InstallIdentity}}import '@open-core/identity';{{end}}

// Register your controllers - OpenCore scans decorators automatically
// Example: import './my-feature.server';

Server.init({
    mode: 'CORE',
}).catch( (error: unknown) => {
    console.error(error)
}).then(()=> {
    console.log('{{.ProjectName}} server initialized!')
})