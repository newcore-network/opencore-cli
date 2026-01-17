// OpenCore Framework - Server Entry Point
import { Server } from '@open-core/framework/server';
{{if .InstallIdentity}}import '@open-core/identity';{{end}}

// Register your controllers - OpenCore scans decorators automatically
// Example imports:
// import './modules/banking/server/banking.controller';

Server.init({
    mode: 'CORE',
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ProjectName}} server initialized!')
})