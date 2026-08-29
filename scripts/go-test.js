#!/usr/bin/env node

const { spawnSync } = require('node:child_process');

const command = process.platform === 'win32' ? 'go.exe' : 'go';
const result = spawnSync(command, ['test', '-json', './...'], {
  encoding: 'utf8',
  maxBuffer: 64 * 1024 * 1024,
});

if (result.error) {
  console.error(`Could not run Go tests: ${result.error.message}`);
  process.exit(1);
}

const skipped = [];
for (const line of result.stdout.split(/\r?\n/)) {
  if (!line) continue;
  let event;
  try {
    event = JSON.parse(line);
  } catch {
    process.stdout.write(`${line}\n`);
    continue;
  }
  if (event.Output) process.stdout.write(event.Output);
  if (event.Action === 'skip' && event.Test) skipped.push(`${event.Package}: ${event.Test}`);
}
if (result.stderr) process.stderr.write(result.stderr);

if (result.status !== 0) process.exit(result.status || 1);
if (skipped.length > 0) {
  console.error(`Go tests may not be skipped:\n${skipped.map((test) => `- ${test}`).join('\n')}`);
  process.exit(1);
}
