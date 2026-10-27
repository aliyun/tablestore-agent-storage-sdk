const assert = require('node:assert');
const { execFileSync } = require('node:child_process');

execFileSync('npm', ['run', 'clean'], { stdio: 'inherit' });

const packOutput = execFileSync('npm', ['pack', '--dry-run', '--json'], {
  encoding: 'utf8',
  stdio: ['ignore', 'pipe', 'inherit'],
});

const [packInfo] = JSON.parse(packOutput);
const bundledPaths = new Set(packInfo.files.map((file) => file.path));

assert.ok(
  bundledPaths.has('dist/src/index.js'),
  'package tarball should include the ESM root entrypoint',
);
assert.ok(
  bundledPaths.has('dist/cjs/src/index.js'),
  'package tarball should include the CommonJS root entrypoint',
);
assert.ok(
  bundledPaths.has('dist/src/node.js'),
  'package tarball should include the ESM node entrypoint',
);
assert.ok(
  bundledPaths.has('dist/cjs/src/node.js'),
  'package tarball should include the CommonJS node entrypoint',
);
