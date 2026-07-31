const assert = require('node:assert/strict');
const test = require('node:test');

const { getBinaryPath, getPlatformPackage, run } = require('../platform');

test('selects the native package for every supported platform', () => {
  assert.deepEqual(getPlatformPackage('linux', 'x64'), {
    packageName: '@open-core/cli-linux-x64',
    binary: 'opencore',
  });
  assert.deepEqual(getPlatformPackage('linux', 'arm64'), {
    packageName: '@open-core/cli-linux-arm64',
    binary: 'opencore',
  });
  assert.deepEqual(getPlatformPackage('darwin', 'x64'), {
    packageName: '@open-core/cli-darwin-x64',
    binary: 'opencore',
  });
  assert.deepEqual(getPlatformPackage('darwin', 'arm64'), {
    packageName: '@open-core/cli-darwin-arm64',
    binary: 'opencore',
  });
  assert.deepEqual(getPlatformPackage('win32', 'x64'), {
    packageName: '@open-core/cli-win32-x64',
    binary: 'opencore.exe',
  });
});

test('rejects unsupported platforms', () => {
  assert.equal(getPlatformPackage('freebsd', 'x64'), null);
  assert.throws(() => getBinaryPath('freebsd', 'x64'), /Unsupported platform/);
});

test('explains when the optional platform package is missing', () => {
  assert.throws(
    () => getBinaryPath('linux', 'x64', () => { throw new Error('not found'); }),
    /Missing optional dependency @open-core\/cli-linux-x64/,
  );
});

test('passes all arguments and propagates the child exit code', () => {
  let invocation;
  const status = run(['build', '--output=plain'], {
    platform: 'linux',
    arch: 'x64',
    resolve: (request) => {
      assert.equal(request, '@open-core/cli-linux-x64/bin/opencore');
      return '/packages/opencore';
    },
    spawnSync: (binary, args, options) => {
      invocation = { binary, args, options };
      return { status: 42 };
    },
  });

  assert.equal(status, 42);
  assert.deepEqual(invocation, {
    binary: '/packages/opencore',
    args: ['build', '--output=plain'],
    options: { stdio: 'inherit', windowsHide: true },
  });
});

test('returns an error code without spawning when the platform package is absent', () => {
  let message = '';
  const status = run([], {
    platform: 'win32',
    arch: 'x64',
    resolve: () => { throw new Error('not found'); },
    spawnSync: () => { throw new Error('must not run'); },
    writeError: (value) => { message = value; },
  });

  assert.equal(status, 1);
  assert.match(message, /@open-core\/cli-win32-x64/);
});
