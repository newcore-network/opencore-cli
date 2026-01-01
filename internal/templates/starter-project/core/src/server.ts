// OpenCore Framework - Server Entry Point
import { Server } from '@open-core/framework';
{{if .InstallIdentity}}import '@open-core/identity';{{end}}

// Register your controllers - OpenCore scans decorators automatically
// Example imports based on your architecture:
// import './modules/banking/server/banking.controller';
// import './features/jobs';

Server.init({
    mode: 'CORE',
    features: {
        netEvents: {enabled: true},
        commands: {enabled: true},
        players: {enabled: true},
        exports: {enabled: true},
        principal: {enabled: true}
    },
    devMode: {
        enabled: true
    }
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ProjectName}} server initialized!')
})