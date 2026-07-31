#!/usr/bin/env node

const { spawnSync } = require('node:child_process');
const path = require('node:path');

const root = path.resolve(__dirname, '..');
const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm';

const checks = [
  [process.execPath, ['scripts/version.js', 'check']],
  [npmCommand, ['test']],
  ['go', ['test', './...']],
  ['go', ['vet', './...']],
  [npmCommand, ['pack', '--dry-run']],
];

for (const [command, args] of checks) {
  const result = spawnSync(command, args, { cwd: root, stdio: 'inherit' });
  if (result.error) {
    console.error(`Release check failed to run ${command}: ${result.error.message}`);
    process.exitCode = 1;
    break;
  }
  if (result.status !== 0) {
    process.exitCode = result.status || 1;
    break;
  }
}
