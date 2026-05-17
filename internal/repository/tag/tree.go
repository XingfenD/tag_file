package tag

// Tree traversal methods removed — callers now use in-memory adjacency maps
// via List().Offset(0).Limit(0).Exec() to load all tags in a single query,
// then traverse parent/child relationships without further DB calls.
// See internal/service/tag/service.go: ListTagTree, GetDescendantIDs, getDescendantIDsInMem.
