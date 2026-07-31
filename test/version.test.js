const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');

const {
  PLATFORM_PACKAGES,
  checkTag,
  isValidVersion,
  setVersion,
  validateVersions,
} = require('../scripts/version');

function writeJSON(root, file, value) {
  const destination = path.join(root, file);
  fs.mkdirSync(path.dirname(destination), { recursive: true });
  fs.writeFileSync(destination, `${JSON.stringify(value, null, 2)}\n`);
}

function createFixture(version = '2.0.0-beta.1') {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), 'opencore-version-'));
  const optionalDependencies = Object.fromEntries(PLATFORM_PACKAGES.map(({ name }) => [name, version]));
  writeJSON(root, 'package.json', {
    name: '@open-core/cli',
    version,
    optionalDependencies,
  });
  for (const platform of PLATFORM_PACKAGES) {
    writeJSON(root, `${platform.directory}/package.json`, {
      name: platform.name,
      version,
    });
  }
  return root;
}

function readJSON(root, file) {
  return JSON.parse(fs.readFileSync(path.join(root, file), 'utf8'));
}

test('accepts stable and prerelease SemVer versions', () => {
  assert.equal(isValidVersion('2.0.0'), true);
  assert.equal(isValidVersion('2.0.0-beta.2'), true);
  assert.equal(isValidVersion('2.0.0-beta.2+build.4'), true);
});

test('rejects invalid versions and a v prefix in set', () => {
  const root = createFixture();
  assert.equal(isValidVersion('2.0'), false);
  assert.equal(isValidVersion('v2.0.0'), false);
  assert.throws(() => setVersion('v2.0.0', root), /without a v prefix/);
  assert.throws(() => setVersion('2.0', root), /Invalid version/);
});

test('set updates every package and optional dependency with final newlines', () => {
  const root = createFixture();
  const version = '2.0.0-beta.2';
  const modified = setVersion(version, root);

  assert.equal(modified.length, 6);
  assert.equal(validateVersions(root), version);
  for (const file of ['package.json', ...PLATFORM_PACKAGES.map(({ directory }) => `${directory}/package.json`)]) {
    assert.equal(fs.readFileSync(path.join(root, file), 'utf8').endsWith('\n'), true);
  }
  const main = readJSON(root, 'package.json');
  for (const { name } of PLATFORM_PACKAGES) {
    assert.equal(main.optionalDependencies[name], version);
  }
});

test('detects a mismatched native package version', () => {
  const root = createFixture();
  const file = `${PLATFORM_PACKAGES[0].directory}/package.json`;
  const pkg = readJSON(root, file);
  pkg.version = '2.0.0-beta.2';
  writeJSON(root, file, pkg);

  assert.throws(() => validateVersions(root), /Version mismatch:[\s\S]*cli-linux-x64: 2.0.0-beta.2/);
});

test('detects missing and unknown optional dependencies', () => {
  const root = createFixture();
  const main = readJSON(root, 'package.json');
  delete main.optionalDependencies[PLATFORM_PACKAGES[0].name];
  main.optionalDependencies['@open-core/cli-old-platform'] = main.version;
  writeJSON(root, 'package.json', main);

  assert.throws(() => validateVersions(root), /Missing optional dependency:[\s\S]*Unknown optional dependency/);
});

test('detects an undeclared native package directory', () => {
  const root = createFixture();
  writeJSON(root, 'npm/freebsd-x64/package.json', {
    name: '@open-core/cli-freebsd-x64',
    version: '2.0.0-beta.1',
  });

  assert.throws(() => validateVersions(root), /Native package is not declared: npm\/freebsd-x64/);
});

test('check-tag accepts a matching tag and rejects a different version', () => {
  const root = createFixture();
  assert.equal(checkTag('v2.0.0-beta.1', root), '2.0.0-beta.1');
  assert.throws(() => checkTag('v2.0.0-beta.2', root), /Tag version mismatch/);
});
