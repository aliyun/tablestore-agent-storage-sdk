import type {
  DeleteItemRequest,
  GetItemRequest,
  ItemPrecondition,
  ItemScopedRequest,
  UpdateItemRequest,
} from '../../src/index.js';

type HasKey<T, K extends PropertyKey> = K extends keyof T ? true : false;
type IsRequired<T, K extends keyof T> = {} extends Pick<T, K> ? false : true;

const scope = { appId: 'a', tenantId: 't', agentId: 'ag', runId: 'r' };
const precondition: ItemPrecondition = { expectedVersionId: 'mv1' };
const byId: GetItemRequest = { memoryStoreName: 'store', scope, itemId: 'm1' };
const byPath: GetItemRequest = { memoryStoreName: 'store', scope, path: '/a.md' };
const update: UpdateItemRequest = {
  memoryStoreName: 'store',
  scope,
  itemId: 'm1',
  content: 'new',
  path: '/b.md',
  precondition,
};
const exactDelete: DeleteItemRequest = {
  memoryStoreName: 'store',
  scope,
  itemId: 'm1',
  precondition,
};
const rangedDelete: DeleteItemRequest = {
  memoryStoreName: 'store',
  scope: { appId: 'a', tenantId: 't', agentId: '*', runId: '*' },
  pathPrefix: '/notes/',
};

const updateItemIdIsRequired: IsRequired<UpdateItemRequest, 'itemId'> = true;
const preconditionVersionIsRequired: IsRequired<ItemPrecondition, 'expectedVersionId'> = true;
const scopedMemoryStoreNameIsRequired: IsRequired<ItemScopedRequest, 'memoryStoreName'> = true;
const scopedScopeIsRequired: IsRequired<ItemScopedRequest, 'scope'> = true;
const updateHasNoNewPath: HasKey<UpdateItemRequest, 'newPath'> = false;
const updateHasNoExpectedSha: HasKey<UpdateItemRequest, 'expectedSha256'> = false;
const deleteHasNoExpectedSha: HasKey<DeleteItemRequest, 'expectedSha256'> = false;

void [
  byId,
  byPath,
  update,
  exactDelete,
  rangedDelete,
  updateItemIdIsRequired,
  preconditionVersionIsRequired,
  scopedMemoryStoreNameIsRequired,
  scopedScopeIsRequired,
  updateHasNoNewPath,
  updateHasNoExpectedSha,
  deleteHasNoExpectedSha,
];
