fx_version 'cerulean'
game '{{.ManifestGame}}'
{{if .AddRedMWarning}}
rdr3_warning 'I acknowledge that this is a prerelease build of RedM, and I am aware my resources *will* become incompatible once RedM ships.'
{{end}}

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

