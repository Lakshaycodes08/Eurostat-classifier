#!/usr/bin/env node
'use strict';

const { spawnSync } = require('child_process');

const PLATFORM_PACKAGES = {
  'darwin-arm64': 'swytchcode-cli-darwin-arm64',
  'darwin-x64':   'swytchcode-cli-darwin-x64',
  'linux-arm64':  'swytchcode-cli-linux-arm64',
  'linux-x64':    'swytchcode-cli-linux-x64',
  'win32-arm64':  'swytchcode-cli-win32-arm64',
  'win32-x64':    'swytchcode-cli-win32-x64',
};

const platform = process.platform;
const arch = process.arch;
const key = `${platform}-${arch}`;
const pkg = PLATFORM_PACKAGES[key];

if (!pkg) {
  process.stderr.write(
    `swytchcode: unsupported platform "${key}".\n` +
    `Supported: ${Object.keys(PLATFORM_PACKAGES).join(', ')}\n` +
    `Install via curl instead: https://cli.swytchcode.com\n`
  );
  process.exit(1);
}

const isWindows = platform === 'win32';
const binName = isWindows ? 'swytchcode.exe' : 'swytchcode';

let binPath;
try {
  binPath = require.resolve(`${pkg}/bin/${binName}`);
} catch (_) {
  process.stderr.write(
    `swytchcode: could not find the binary for "${key}".\n` +
    `Try reinstalling: npm install -g swytchcode\n`
  );
  process.exit(1);
}

const result = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });

if (result.error) {
  process.stderr.write(`swytchcode: failed to launch binary: ${result.error.message}\n`);
  process.exit(1);
}

process.exit(result.status ?? 1);
