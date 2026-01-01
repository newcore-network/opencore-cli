import { Client } from '@open-core/framework';

Client.init({
    mode: 'RESOURCE'
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ResourceName}} client initialized!')
})

console.log('{{.ResourceName}} client loaded');

