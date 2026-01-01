// {{.StandaloneName}} - Server Side
// This is a standalone resource (no OpenCore Framework dependency)

console.log('[{{.StandaloneName}}] Server started');

// Register your commands
RegisterCommand('{{.StandaloneName}}', (source: number, args: string[]) => {
    console.log(`[{{.StandaloneName}}] Command executed by ${source}`);
}, false);

// Your server logic here
