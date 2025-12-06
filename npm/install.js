#!/usr/bin/env node

const https = require('https');
const fs = require('fs');
const path = require('path');
const { getPlatform, getBinaryName } = require('./binary');

const VERSION = require('./package.json').version;
const GITHUB_REPO = 'newcore-network/opencore-cli';

async function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    
    https.get(url, (response) => {
      if (response.statusCode === 302 || response.statusCode === 301) {
        // Follow redirect
        return download(response.headers.location, dest).then(resolve).catch(reject);
      }
      
      if (response.statusCode !== 200) {
        reject(new Error(`Failed to download: ${response.statusCode} ${response.statusMessage}`));
        return;
      }
      
      const totalBytes = parseInt(response.headers['content-length'], 10);
      let downloadedBytes = 0;
      
      response.on('data', (chunk) => {
        downloadedBytes += chunk.length;
        const percent = ((downloadedBytes / totalBytes) * 100).toFixed(1);
        process.stdout.write(`\rDownloading OpenCore CLI... ${percent}%`);
      });
      
      response.pipe(file);
      
      file.on('finish', () => {
        file.close();
        console.log('\nDownload complete!');
        resolve();
      });
    }).on('error', (err) => {
      fs.unlink(dest, () => {});
      reject(err);
    });
  });
}

async function install() {
  try {
    console.log('Installing OpenCore CLI...');
    
    const platform = getPlatform();
    if (!platform) {
      throw new Error(`Unsupported platform: ${process.platform} ${process.arch}`);
    }
    
    const binaryName = getBinaryName();
    const binDir = path.join(__dirname, 'bin');
    
    // Create bin directory
    if (!fs.existsSync(binDir)) {
      fs.mkdirSync(binDir, { recursive: true });
    }
    
    const binaryPath = path.join(binDir, binaryName);
    
    // Download binary from GitHub releases
    const downloadUrl = `https://github.com/${GITHUB_REPO}/releases/download/v${VERSION}/opencore-${platform}${platform.startsWith('windows') ? '.exe' : ''}`;
    
    console.log(`Downloading from: ${downloadUrl}`);
    await download(downloadUrl, binaryPath);
    
    // Make binary executable on Unix systems
    if (process.platform !== 'win32') {
      fs.chmodSync(binaryPath, 0o755);
    }
    
    console.log('✓ OpenCore CLI installed successfully!');
    console.log(`Run 'opencore --version' to verify installation.`);
  } catch (error) {
    console.error('Failed to install OpenCore CLI:', error.message);
    console.error('\nYou can manually download the binary from:');
    console.error(`https://github.com/${GITHUB_REPO}/releases/tag/v${VERSION}`);
    process.exit(1);
  }
}

install();

