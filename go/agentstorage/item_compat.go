package agentstorage

import "context"

// resolveItemGuard translates a legacy SHA guard to the current resource
// version. The write still uses a server-side precondition, so a concurrent
// mutation between this read and the write cannot silently defeat the guard.
func (c *Client) resolveItemGuard(ctx context.Context, scoped ItemScopedRequest, itemID, path, sha string, precondition *ItemPrecondition, view ItemView) (*Item, error) {
	if itemID == "" && path == "" {
		return nil, newClientErrorWithCause(ClientErrorKindRequestBuild, nil, "legacy item request requires an exact itemId or path")
	}
	if itemID != "" {
		path = ""
	}
	item, err := c.GetItem(ctx, &GetItemRequest{ItemScopedRequest: scoped, ItemID: itemID, Path: path, View: view})
	if err != nil {
		return nil, err
	}
	if item.ItemID == "" {
		return nil, newClientErrorWithCause(ClientErrorKindResponseSemantic, nil, "GetItem returned no itemId for the legacy request")
	}
	if sha != "" {
		if item.ContentSHA256 != sha {
			return nil, newClientErrorWithCause(ClientErrorKindRequestBuild, nil, "ExpectedSHA256 does not match the current item")
		}
		if item.LatestVersionID == "" {
			return nil, newClientErrorWithCause(ClientErrorKindResponseSemantic, nil, "GetItem returned no latestVersionId for the SHA guard")
		}
	}
	if precondition != nil && precondition.ExpectedVersionID != item.LatestVersionID {
		return nil, newClientErrorWithCause(ClientErrorKindRequestBuild, nil, "Precondition does not match the current item version")
	}
	return item, nil
}

// SearchItems searches the summaries of managed files.
func (c *Client) SearchItems(ctx context.Context, req *SearchItemsRequest) (*SearchItemsResponse, error) {
	if req == nil {
		req = &SearchItemsRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[SearchItemsResponse](c, ctx, "SearchItems", &r)
}

// DeleteItemVersions starts asynchronous removal of historical versions after
// the current items in the requested scope have been removed.
func (c *Client) DeleteItemVersions(ctx context.Context, req *DeleteItemVersionsRequest) (*DeleteItemVersionsResponse, error) {
	if req == nil {
		req = &DeleteItemVersionsRequest{}
	}
	r := *req
	r.setDefaultType()
	return do[DeleteItemVersionsResponse](c, ctx, "DeleteItemVersions", &r)
}
