// {{.StandaloneName}} - Server Side
import { Server } from '@open-core/framework/server';

// Bootstrap the standalone server
// Standalone mode enables guards and decorators without depending on a core resource
Server.init({
  mode: 'STANDALONE',
}).catch( (error: unknown) => {
    console.error(error)
}).then(()=> {
    console.log('[{{.StandaloneName}}] Server initialized in STANDALONE mode');
});

// Your server logic here
