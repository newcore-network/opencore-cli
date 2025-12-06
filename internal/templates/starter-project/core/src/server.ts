// OpenCore Framework - Server Entry Point
{{if .InstallIdentity}}import '@open-core/identity';{{end}}

// Import your controllers and services here
// OpenCore will automatically scan and register them via decorators
// Example:
// import './features/banking';
// import './modules/jobs';

console.log('{{.ProjectName}} server initialized!');

