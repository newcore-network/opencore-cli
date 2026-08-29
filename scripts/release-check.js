#!/usr/bin/env node

const fs = require('node:fs');
const { spawnSync } = require('node:child_process');
const path = require('node:path');
const { checkTag, validateVersions } = require('./version');

const root = path.resolve(__dirname, '..');
const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm';

try {
  const tag = process.argv[2] || process.env.GITHUB_REF_NAME || `v${validateVersions(root)}`;
  checkTag(tag, root);
  const releaseNotes = fs.readFileSync(path.join(root, 'RELEASE.md'), 'utf8');
  const escapedTag = tag.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  if (!new RegExp(`^#{1,6}\\s+.*${escapedTag}(?:\\s|$)`, 'mi').test(releaseNotes)) {
    throw new Error(`RELEASE.md does not contain a heading for ${tag}`);
  }
} catch (error) {
  console.error(`Release preflight failed: ${error.message}`);
  process.exit(1);
}

const checks = [
  [npmCommand, ['test']],
  [process.execPath, ['scripts/go-test.js']],
  ['go', ['vet', './...']],
  [npmCommand, ['pack', '--dry-run']],
];

for (const [command, args] of checks) {
  const result = spawnSync(command, args, { cwd: root, stdio: 'inherit' });
  if (result.error) {
    console.error(`Release check failed to run ${command}: ${result.error.message}`);
    process.exit(1);
  }
  if (result.status !== 0) process.exit(result.status || 1);
}
