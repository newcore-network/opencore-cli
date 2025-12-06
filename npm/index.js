#!/usr/bin/env node

const { getBinaryPath } = require('./binary');
const { spawn } = require('child_process');

const binaryPath = getBinaryPath();

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  windowsHide: true
});

child.on('exit', (code) => {
  process.exit(code || 0);
});

process.on('SIGINT', () => {
  child.kill('SIGINT');
});

process.on('SIGTERM', () => {
  child.kill('SIGTERM');
});

