fx_version 'cerulean'
game 'gta5'

name '{{.ProjectName}}-core'
description 'OpenCore server core'
author 'Your Name'
version '1.0.0'

server_scripts {
    'dist/server/**/*.js'
}

client_scripts {
    'dist/client/**/*.js'
}

