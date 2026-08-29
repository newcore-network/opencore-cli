#!/usr/bin/env node

const fs = require('node:fs');
const path = require('node:path');

const MAIN_PACKAGE = 'package.json';
const PLATFORM_PACKAGES = [
  { name: '@open-core/cli-linux-x64', directory: 'npm/linux-x64' },
  { name: '@open-core/cli-linux-arm64', directory: 'npm/linux-arm64' },
  { name: '@open-core/cli-darwin-x64', directory: 'npm/darwin-x64' },
  { name: '@open-core/cli-darwin-arm64', directory: 'npm/darwin-arm64' },
  { name: '@open-core/cli-win32-x64', directory: 'npm/win32-x64' },
];

const SEMVER_PATTERN = /^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-((?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9]\d*|\d*[A-Za-z-][0-9A-Za-z-]*))*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/;

function isValidVersion(version) {
  return typeof version === 'string' && SEMVER_PATTERN.test(version);
}

function packageDefinitions() {
  return [
    { name: '@open-core/cli', file: MAIN_PACKAGE },
    ...PLATFORM_PACKAGES.map((platform) => ({
      name: platform.name,
      file: path.join(platform.directory, 'package.json'),
    })),
  ];
}

function readJSON(root, file) {
  return JSON.parse(fs.readFileSync(path.join(root, file), 'utf8'));
}

function writeJSON(root, file, value) {
  fs.writeFileSync(path.join(root, file), `${JSON.stringify(value, null, 2)}\n`);
}

function expectedOptionalDependencies() {
  return new Set(PLATFORM_PACKAGES.map((platform) => platform.name));
}

function listNativePackageDirectories(root) {
  const npmDirectory = path.join(root, 'npm');
  if (!fs.existsSync(npmDirectory)) {
    return [];
  }

  return fs.readdirSync(npmDirectory, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && fs.existsSync(path.join(npmDirectory, entry.name, 'package.json')))
    .map((entry) => `npm/${entry.name}`)
    .sort();
}

function validateVersions(root = process.cwd()) {
  const definitions = packageDefinitions();
  const packages = definitions.map((definition) => ({
    ...definition,
    value: readJSON(root, definition.file),
  }));
  const main = packages[0].value;
  const expectedNames = expectedOptionalDependencies();
  const expectedDirectories = new Set(PLATFORM_PACKAGES.map((platform) => platform.directory));
  const errors = [];
  const versionLines = packages.map((pkg) => `- ${pkg.value.name || pkg.name}: ${pkg.value.version}`);
  const versions = new Set(packages.map((pkg) => pkg.value.version));

  if (versions.size !== 1) {
    errors.push(`Version mismatch:\n${versionLines.join('\n')}`);
  }

  for (const pkg of packages) {
    if (pkg.value.name !== pkg.name) {
      errors.push(`Package name mismatch in ${pkg.file}: ${JSON.stringify(pkg.value.name)}, expected ${pkg.name}`);
    }
    if (!isValidVersion(pkg.value.version)) {
      errors.push(`Invalid version in ${pkg.file}: ${JSON.stringify(pkg.value.version)}`);
    }
  }

  const optionalDependencies = main.optionalDependencies || {};
  for (const name of expectedNames) {
    if (!(name in optionalDependencies)) {
      errors.push(`Missing optional dependency: ${name}`);
    } else if (optionalDependencies[name] !== main.version) {
      errors.push(`Optional dependency version mismatch: ${name} is ${optionalDependencies[name]}, expected ${main.version}`);
    }
  }
  for (const name of Object.keys(optionalDependencies)) {
    if (!expectedNames.has(name)) {
      errors.push(`Unknown optional dependency: ${name}`);
    }
  }

  const discoveredDirectories = new Set(listNativePackageDirectories(root));
  for (const directory of discoveredDirectories) {
    if (!expectedDirectories.has(directory)) {
      errors.push(`Native package is not declared: ${directory}`);
    }
  }
  for (const directory of expectedDirectories) {
    if (!discoveredDirectories.has(directory)) {
      errors.push(`Native package directory is missing: ${directory}`);
    }
  }

  if (errors.length > 0) {
    throw new Error(errors.join('\n'));
  }

  return main.version;
}

function setVersion(version, root = process.cwd()) {
  if (typeof version !== 'string' || version.startsWith('v') || !isValidVersion(version)) {
    throw new Error(`Invalid version ${JSON.stringify(version)}. Use a SemVer version without a v prefix.`);
  }

  const modified = [];
  for (const definition of packageDefinitions()) {
    const pkg = readJSON(root, definition.file);
    let changed = pkg.version !== version;
    pkg.version = version;
    if (definition.file === MAIN_PACKAGE) {
      const optionalDependencies = { ...pkg.optionalDependencies };
      for (const platform of PLATFORM_PACKAGES) {
        if (optionalDependencies[platform.name] !== version) {
          changed = true;
        }
        optionalDependencies[platform.name] = version;
      }
      pkg.optionalDependencies = optionalDependencies;
    }
    if (changed) {
      writeJSON(root, definition.file, pkg);
      modified.push(definition.file);
    }
  }

  return modified;
}

function checkTag(tag, root = process.cwd()) {
  if (typeof tag !== 'string' || !tag.startsWith('v') || !isValidVersion(tag.slice(1))) {
    throw new Error(`Invalid tag ${JSON.stringify(tag)}. Use v followed by a SemVer version.`);
  }

  const version = validateVersions(root);
  if (version !== tag.slice(1)) {
    throw new Error(`Tag version mismatch: ${tag} does not match package version ${version}`);
  }
  return version;
}

function run(argv = process.argv.slice(2), root = process.cwd()) {
  const [command, value] = argv;
  switch (command) {
    case 'set': {
      if (!value || argv.length !== 2) {
        throw new Error('Usage: version.js set <version>');
      }
      const modified = setVersion(value, root);
      if (modified.length === 0) {
        console.log(`All package versions already use ${value}.`);
      } else {
        console.log(`Set version ${value} in:`);
        for (const file of modified) {
          console.log(`- ${file}`);
        }
      }
      return;
    }
    case 'check':
      if (argv.length !== 1) {
        throw new Error('Usage: version.js check');
      }
      console.log(`Version check passed: ${validateVersions(root)}`);
      return;
    case 'check-tag':
      if (!value || argv.length !== 2) {
        throw new Error('Usage: version.js check-tag <tag>');
      }
      console.log(`Tag version check passed: ${checkTag(value, root)}`);
      return;
    default:
      throw new Error('Usage: version.js <set|check|check-tag> [version-or-tag]');
  }
}

if (require.main === module) {
  try {
    run();
  } catch (error) {
    console.error(`Version check failed: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = {
  MAIN_PACKAGE,
  PLATFORM_PACKAGES,
  checkTag,
  isValidVersion,
  run,
  setVersion,
  validateVersions,
};
