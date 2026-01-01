import { Client } from '@open-core/framework';

Client.init({
    mode: 'CORE'
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('{{.ResourceName}} client initialized!')
})

console.log('{{.ResourceName}} client loaded');

