const assert = require('node:assert');

const core = require('@tablestore/agent-storage');
const nodeEntrypoint = require('@tablestore/agent-storage/node');

assert.strictEqual(typeof core.AgentStorageClient, 'function');
assert.strictEqual(typeof nodeEntrypoint.NodeAgentStorageClient, 'function');
