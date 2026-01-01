fx_version 'cerulean'
game 'gta5'

name '{{.StandaloneName}}'
author 'Your Name'
version '1.0.0'

server_scripts {
    'server.js'
}
{{if .HasClient}}
client_scripts {
    'client.js'
}
{{end}}
{{if .HasNUI}}
ui_page 'ui/index.html'

files {
    'ui/**/*'
}
{{end}}
