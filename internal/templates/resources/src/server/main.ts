import { Server } from '@open-core/framework/server';

// Bootstrap the resource server
Server.init({
  mode: 'RESOURCE',
  coreResourceName: 'core',
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ResourceName}} server initialized!')
})