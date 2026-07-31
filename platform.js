const { spawnSync } = require('node:child_process');

const platforms = {
  'darwin-arm64': {
    packageName: '@open-core/cli-darwin-arm64',
    binary: 'opencore',
  },
  'darwin-x64': {
    packageName: '@open-core/cli-darwin-x64',
    binary: 'opencore',
  },
  'linux-arm64': {
    packageName: '@open-core/cli-linux-arm64',
    binary: 'opencore',
  },
  'linux-x64': {
    packageName: '@open-core/cli-linux-x64',
    binary: 'opencore',
  },
  'win32-x64': {
    packageName: '@open-core/cli-win32-x64',
    binary: 'opencore.exe',
  },
};

function getPlatformPackage(platform = process.platform, arch = process.arch) {
  return platforms[`${platform}-${arch}`] || null;
}

function getBinaryPath(platform = process.platform, arch = process.arch, resolve = require.resolve) {
  const target = getPlatformPackage(platform, arch);
  if (!target) {
    throw new Error(`Unsupported platform: ${platform} ${arch}. OpenCore CLI supports Linux (x64, arm64), macOS (x64, arm64), and Windows (x64).`);
  }

  try {
    return resolve(`${target.packageName}/bin/${target.binary}`);
  } catch (error) {
    throw new Error(`Missing optional dependency ${target.packageName} for ${platform} ${arch}. Reinstall @open-core/cli on this platform.`);
  }
}

function run(args, options = {}) {
  const platform = options.platform || process.platform;
  const arch = options.arch || process.arch;
  const resolve = options.resolve || require.resolve;
  const execute = options.spawnSync || spawnSync;
  const writeError = options.writeError || ((message) => console.error(message));

  let binaryPath;
  try {
    binaryPath = getBinaryPath(platform, arch, resolve);
  } catch (error) {
    writeError(`OpenCore CLI: ${error.message}`);
    return 1;
  }

  const result = execute(binaryPath, args, {
    stdio: 'inherit',
    windowsHide: true,
  });
  if (result.error) {
    writeError(`OpenCore CLI: failed to execute ${binaryPath}: ${result.error.message}`);
    return 1;
  }

  return typeof result.status === 'number' ? result.status : 1;
}

module.exports = { getPlatformPackage, getBinaryPath, run };
