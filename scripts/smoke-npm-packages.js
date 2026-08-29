#!/usr/bin/env node

const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const archiveDirectory = path.resolve(process.argv[2] || 'dist/npm');
const expectedVersion = require('../package.json').version;
const archives = fs.readdirSync(archiveDirectory)
  .filter((file) => file.endsWith('.tgz'))
  .map((file) => path.join(archiveDirectory, file));

if (archives.length !== 6) {
  throw new Error(`Expected 6 npm tarballs in ${archiveDirectory}, found ${archives.length}`);
}

const installDirectory = fs.mkdtempSync(path.join(os.tmpdir(), 'opencore-npm-smoke-'));
try {
  fs.writeFileSync(path.join(installDirectory, 'package.json'), '{"private":true}\n');
  const npm = process.platform === 'win32' ? 'npm.cmd' : 'npm';
  const install = spawnSync(npm, [
    'install', '--force', '--ignore-scripts', '--no-audit', '--no-fund', '--package-lock=false', ...archives,
  ], { cwd: installDirectory, stdio: 'inherit' });
  if (install.status !== 0) process.exit(install.status || 1);

  const main = require(path.join(installDirectory, 'node_modules/@open-core/cli/package.json'));
  if (main.version !== expectedVersion) {
    throw new Error(`Installed main package is ${main.version}, expected ${expectedVersion}`);
  }
  for (const name of Object.keys(main.optionalDependencies)) {
    const platformPackage = require(path.join(installDirectory, 'node_modules', name, 'package.json'));
    if (platformPackage.version !== expectedVersion) {
      throw new Error(`${name} is ${platformPackage.version}, expected ${expectedVersion}`);
    }
  }

  const executable = process.platform === 'win32'
    ? path.join(installDirectory, 'node_modules/.bin/opencore.cmd')
    : path.join(installDirectory, 'node_modules/.bin/opencore');
  const smoke = spawnSync(executable, ['--version'], { stdio: 'inherit' });
  if (smoke.status !== 0) process.exit(smoke.status || 1);
} finally {
  fs.rmSync(installDirectory, { recursive: true, force: true });
}
