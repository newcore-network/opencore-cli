const path = require('path');
const os = require('os');
const fs = require('fs');

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();
  
  if (platform === 'win32') {
    return arch === 'x64' ? 'windows-amd64' : null;
  }
  
  if (platform === 'darwin') {
    if (arch === 'arm64') return 'darwin-arm64';
    if (arch === 'x64') return 'darwin-amd64';
    return null;
  }
  
  if (platform === 'linux') {
    return arch === 'x64' ? 'linux-amd64' : null;
  }
  
  return null;
}

function getBinaryName() {
  const platform = os.platform();
  return platform === 'win32' ? 'opencore.exe' : 'opencore';
}

function getBinaryPath() {
  const platform = getPlatform();
  if (!platform) {
    throw new Error(`Unsupported platform: ${os.platform()} ${os.arch()}`);
  }
  
  const binaryName = getBinaryName();
  const binaryPath = path.join(__dirname, 'bin', binaryName);
  
  if (!fs.existsSync(binaryPath)) {
    throw new Error(
      `Binary not found at ${binaryPath}. ` +
      `This might be an installation issue. Please try reinstalling @open-core/cli.`
    );
  }
  
  return binaryPath;
}

module.exports = {
  getPlatform,
  getBinaryName,
  getBinaryPath
};

