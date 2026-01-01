import { Server } from '@open-core/framework';

// Bootstrap the resource server
Server.init({
  mode: 'RESOURCE',
  coreResourceName: 'core',
  features: {
    commands: { enabled: true,  provider: 'core'},
    players: {enabled: true, provider: 'core'},
    netEvents: {enabled: true}
  }
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ResourceName}} server initialized!')
})