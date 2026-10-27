const { mkdirSync, writeFileSync } = require('node:fs');
const { join } = require('node:path');

const cjsDistDir = join(__dirname, '..', 'dist', 'cjs');

mkdirSync(cjsDistDir, { recursive: true });
writeFileSync(
  join(cjsDistDir, 'package.json'),
  `${JSON.stringify({ type: 'commonjs' }, null, 2)}\n`,
);
