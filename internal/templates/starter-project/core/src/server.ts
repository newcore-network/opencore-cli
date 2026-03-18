// OpenCore Framework - Server Entry Point
import { Server } from '@open-core/framework/server'
{{if .InstallIdentity}}import '@open-core/identity'{{end}}

// OpenCore scans decorators automatically when is Marked as Controller(), if not, import here
// Example: import './my-feature.client';

Server.init({
    mode: 'CORE',
}).catch( (error: unknown) => {
    console.error(error)
}).then(()=> {
    console.log('{{.ProjectName}} server initialized!')
})