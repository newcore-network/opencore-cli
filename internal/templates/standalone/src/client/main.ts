import { Client } from '@open-core/framework/client';

// Bootstrap the standalone client
Client.init({
  mode: 'STANDALONE',
}).catch( error => {
    console.error(error)
}).then(()=> {
    console.log('[{{.StandaloneName}}] Client initialized in STANDALONE mode');
});

// Your client logic here
