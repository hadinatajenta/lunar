package application

import (
	"errors"
	"strings"
	"testing"

	"lunar/backend/internal/postman/domain"
)

func normalizedCollection(t *testing.T, raw string, service string) *domain.Collection {
	t.Helper()
	collection := mustParseCollection(t, raw)
	Normalize(collection, service)
	return collection
}

func TestValidateRefusesACollectionWithNoRequests(t *testing.T) {
	collection := normalizedCollection(t, `{"info":{"name":"atlas API"},"item":[]}`, "atlas")

	err := Validate(collection)
	if !errors.Is(err, domain.ErrCollectionInvalid) {
		t.Fatalf("expected ErrCollectionInvalid, received %v", err)
	}
	if !strings.Contains(err.Error(), "no endpoints found") {
		t.Fatalf("expected a no-endpoints reason, received %q", err.Error())
	}
}

func TestValidateRefusesAHollowRequest(t *testing.T) {
	collection := normalizedCollection(t, `{"item":[{"name":"Hollow","request":{"method":"GET","header":[],"url":{"raw":""}}}]}`, "atlas")

	err := Validate(collection)
	if !errors.Is(err, domain.ErrCollectionInvalid) {
		t.Fatalf("expected ErrCollectionInvalid, received %v", err)
	}
	if !strings.Contains(err.Error(), "url path") {
		t.Fatalf("expected a url path reason, received %q", err.Error())
	}
}

func TestValidateRefusesARequestWithoutAName(t *testing.T) {
	collection := normalizedCollection(t, `{"item":[{"name":"","request":{"method":"GET","header":[],"url":{"path":["v1","user"]}}}]}`, "atlas")

	err := Validate(collection)
	if !errors.Is(err, domain.ErrCollectionInvalid) || !strings.Contains(err.Error(), "no name") {
		t.Fatalf("expected a missing-name refusal, received %v", err)
	}
}

func TestValidateRefusesAQueryParameterWithoutAKey(t *testing.T) {
	collection := normalizedCollection(t, `{"item":[{"name":"List","request":{"method":"GET","header":[],"url":{"path":["v1","user"],"query":[{"value":"1"}]}}}]}`, "atlas")

	err := Validate(collection)
	if !errors.Is(err, domain.ErrCollectionInvalid) || !strings.Contains(err.Error(), "query parameter without a key") {
		t.Fatalf("expected a query key refusal, received %v", err)
	}
}

func TestValidateRefusesABodyFieldWithoutAKey(t *testing.T) {
	collection := normalizedCollection(t, `{"item":[{"name":"Create","request":{"method":"POST","header":[],"url":{"path":["v1","user"]},"body":{"mode":"urlencoded","urlencoded":[{"value":"1"}]}}}]}`, "atlas")

	err := Validate(collection)
	if !errors.Is(err, domain.ErrCollectionInvalid) || !strings.Contains(err.Error(), "body field without a key") {
		t.Fatalf("expected a body key refusal, received %v", err)
	}
}

func TestValidateAcceptsACompleteCollection(t *testing.T) {
	collection := normalizedCollection(t, `{"item":[{"name":"users","item":[{"name":"List levels","request":{"method":"GET","header":[{"key":"Accept","value":"application/json"}],"url":{"path":["v1","user","level"],"query":[{"key":"page","value":"1"}]},"body":{"mode":"raw","raw":"{}"}}}]}]}`, "atlas")

	if err := Validate(collection); err != nil {
		t.Fatalf("a complete collection must pass validation, received %v", err)
	}
	if CountRequests(collection) != 1 {
		t.Fatalf("expected 1 request, received %d", CountRequests(collection))
	}
	if CountFolders(collection) != 1 {
		t.Fatalf("expected 1 folder, received %d", CountFolders(collection))
	}
}

func TestCountFoldersCountsNestedFolders(t *testing.T) {
	collection := normalizedCollection(t, `{"item":[{"name":"v1","item":[{"name":"users","item":[{"name":"List","request":{"method":"GET","header":[],"url":{"path":["v1","user"]}}}]},{"name":"Empty folder","item":[]}]}]}`, "atlas")

	if CountRequests(collection) != 1 {
		t.Fatalf("expected 1 request, received %d", CountRequests(collection))
	}
	if CountFolders(collection) != 2 {
		t.Fatalf("expected 2 folders to contain requests, received %d", CountFolders(collection))
	}
}
