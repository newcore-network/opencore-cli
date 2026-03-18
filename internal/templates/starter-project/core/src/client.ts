// OpenCore Framework - Client Entry Point
import { Client } from '@open-core/framework/client';

// OpenCore scans decorators automatically when is Marked as Controller(), if not, import here
// Example: import './my-feature.client';

Client.init({
    mode: 'CORE'
}).catch( (error: unknown) => {
    console.error(error)
}).then(()=> {
    console.log('{{.ProjectName}} client initialized!')
})
