import { describe, expect, it } from 'vitest';

import * as core from '../../src/index.js';
import * as nodeEntrypoint from '../../src/node.js';

describe('package entrypoints', () => {
  it('exports AgentStorageClient and NodeAgentStorageClient', () => {
    expect(core.AgentStorageClient).toBeTypeOf('function');
    expect(nodeEntrypoint.NodeAgentStorageClient).toBeTypeOf('function');
  });
});
