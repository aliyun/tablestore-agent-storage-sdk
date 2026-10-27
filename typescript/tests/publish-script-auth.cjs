const assert = require('node:assert');
const { readFileSync } = require('node:fs');
const { join } = require('node:path');

const publishScript = readFileSync(join(__dirname, '..', '..', 'scripts', 'publish_npm.sh'), 'utf8');

assert.doesNotMatch(
  publishScript,
  /npm config set _authToken\b/,
  'publish script must not write an unscoped npm _authToken',
);
assert.match(
  publishScript,
  /npm config set ['"]?\/\/registry\.npmjs\.org\/:_authToken['"]?/,
  'publish script should scope the npm auth token to registry.npmjs.org',
);
