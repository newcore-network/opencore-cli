#!/usr/bin/env node

const { getBinaryPath } = require('./binary');
const { spawn } = require('child_process');

// Export config helper for opencore.config.ts
const { defineConfig } = require('./config-helper');
module.exports = { defineConfig };

// Only run the binary if this is being executed directly
if (require.main === module) {
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
}

