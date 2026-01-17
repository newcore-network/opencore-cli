import { Client } from '@open-core/framework/client';

// Bootstrap the resource client
Client.init({
  mode: 'RESOURCE',
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ResourceName}} client initialized!')
})

console.log('{{.ResourceName}} client loaded');

