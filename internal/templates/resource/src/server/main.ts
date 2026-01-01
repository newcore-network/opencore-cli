import { Server } from '@open-core/framework';

// Bootstrap the resource server
Server.init({
    mode: 'CORE',
    features: {
        netEvents: {enabled: true},
        commands: {enabled: true},
        players: {enabled: true}
    },
    devMode: {
        enabled: true
    }
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ResourceName}} server initialized!')
})
console.log('{{.ResourceName}} server loaded');

