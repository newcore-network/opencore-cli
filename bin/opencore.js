#!/usr/bin/env node

const { run } = require('../platform');

process.exitCode = run(process.argv.slice(2));
