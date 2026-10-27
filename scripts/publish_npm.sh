#!/usr/bin/env sh
set -e

cd typescript
npm -v
node -v

npm config set registry https://registry.npmjs.org
npm config set //registry.npmjs.org/:_authToken "${NPM_TOKEN}"
npm ci

echo "dry_run:${dry_run}"

if [ "$dry_run" = "true" ]; then
  npm publish --registry https://registry.npmjs.org --dry-run
else
  npm publish --registry https://registry.npmjs.org
fi
