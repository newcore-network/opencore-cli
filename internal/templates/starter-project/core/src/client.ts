// OpenCore Framework - Client Entry Point
import { Client } from '@open-core/framework/client';

// Register your client controllers - OpenCore scans decorators automatically
// Example: import './my-feature.client';

Client.init({
    mode: 'CORE'
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ProjectName}} client initialized!')
})
