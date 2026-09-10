package comm

import "errors"

// ErrAssetNotFound is returned by Asset and AssetDir when the requested
// embedded asset name does not exist in the bindata map or bintree.
// Callers can use errors.Is(err, ErrAssetNotFound) to detect this case.
var ErrAssetNotFound = errors.New("asset not found")
