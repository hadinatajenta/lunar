package domain

import "errors"

var (
	ErrProviderKeyMissing   = errors.New("no provider key configured")
	ErrServiceNotExportable = errors.New("service cannot be exported")
	ErrCollectionInvalid    = errors.New("collection invalid")
	ErrProviderTimeout      = errors.New("provider timed out")
)
