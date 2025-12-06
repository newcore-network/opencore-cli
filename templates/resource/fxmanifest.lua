fx_version 'cerulean'
game 'gta5'

name '{{.ResourceName}}'
author 'Your Name'
version '1.0.0'

server_scripts {
    'dist/server/**/*.js'
}
{{if .HasClient}}
client_scripts {
    'dist/client/**/*.js'
}
{{end}}
{{if .HasNUI}}
ui_page 'ui/index.html'

files {
    'ui/**/*'
}
{{end}}

