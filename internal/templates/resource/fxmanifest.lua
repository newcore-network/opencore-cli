fx_version 'cerulean'
game 'gta5'

name '{{.ResourceName}}'
author 'Your Name'
version '1.0.0'
node_version '22'

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

dependencies {
    'core'
}


