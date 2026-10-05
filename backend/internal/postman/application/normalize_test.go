package application

import (
	"encoding/json"
	"strings"
	"testing"

	"lunar/backend/internal/postman/domain"
)

func mustParseCollection(t *testing.T, raw string) *domain.Collection {
	t.Helper()
	var collection domain.Collection
	if err := json.Unmarshal([]byte(raw), &collection); err != nil {
		t.Fatalf("cannot decode the fixture collection: %v", err)
	}
	return &collection
}

func TestNormalizeFillsStructuralKeys(t *testing.T) {
	collection := mustParseCollection(t, `{"item":[{"name":"users","item":[{"name":"Get level user","request":{"url":{"raw":"{{baseUrl}}/v1/user/level"}}}]}]}`)

	Normalize(collection, "atlas")

	if collection.Info.Name != "atlas API" {
		t.Fatalf("expected the service name to be derived, received %q", collection.Info.Name)
	}
	if collection.Info.Schema != domain.CollectionSchemaURL {
		t.Fatalf("expected the Postman v2.1 schema URL, received %q", collection.Info.Schema)
	}
	if collection.Variable == nil {
		t.Fatalf("expected the variable array to be present")
	}

	item := collection.Item[0].Item[0]
	request := item.Request
	if request.Method != "GET" {
		t.Fatalf("expected a GET method when the model omitted one, received %q", request.Method)
	}
	if request.Header == nil || len(request.Header) != 0 {
		t.Fatalf("expected an empty header array, received %v", request.Header)
	}
	if request.URL == nil || len(request.URL.Host) != 1 || request.URL.Host[0] != "{{baseUrl}}" {
		t.Fatalf("expected the {{baseUrl}} host variable, received %v", request.URL)
	}
	if strings.Join(request.URL.Path, "/") != "v1/user/level" {
		t.Fatalf("expected the path to be derived from url.raw, received %v", request.URL.Path)
	}
	if string(item.Response) != "[]" {
		t.Fatalf("expected an empty response array, received %s", string(item.Response))
	}
}

func TestNormalizeKeepsModelProvidedValues(t *testing.T) {
	collection := mustParseCollection(t, `{"info":{"name":"Custom API","schema":"custom-schema"},"item":[{"name":"Create user","request":{"method":"POST","header":[{"key":"Content-Type","value":"application/json"}],"url":{"raw":"{{baseUrl}}/v1/user/create","host":["{{baseUrl}}"],"path":["v1","user","create"]},"body":{"mode":"raw","raw":"{}"}}}]}`)

	Normalize(collection, "atlas")

	if collection.Info.Name != "Custom API" || collection.Info.Schema != "custom-schema" {
		t.Fatalf("the normaliser must not overwrite model values, received %q and %q", collection.Info.Name, collection.Info.Schema)
	}

	request := collection.Item[0].Request
	if request.Method != "POST" {
		t.Fatalf("expected the model method to survive, received %q", request.Method)
	}
	if len(request.Header) != 1 || request.Header[0].Key != "Content-Type" {
		t.Fatalf("expected the model headers to survive, received %v", request.Header)
	}
	if request.Body == nil || request.Body.Raw != "{}" {
		t.Fatalf("expected the model body to survive, received %v", request.Body)
	}
}

func TestNormalizeDerivesThePathFromAnAbsoluteURL(t *testing.T) {
	collection := mustParseCollection(t, `{"item":[{"name":"List merchants","request":{"method":"GET","url":{"raw":"http://localhost:8080/api/merchants?page=1#top"}}}]}`)

	Normalize(collection, "merchant-service-laravel")

	path := collection.Item[0].Request.URL.Path
	if strings.Join(path, "/") != "api/merchants" {
		t.Fatalf("expected api/merchants, received %v", path)
	}
}

func TestNormalizeAcceptsAPathString(t *testing.T) {
	collection := mustParseCollection(t, `{"item":[{"name":"List merchants","request":{"method":"GET","url":{"path":"/api/merchants"}}}]}`)

	Normalize(collection, "merchant-service-laravel")

	path := collection.Item[0].Request.URL.Path
	if strings.Join(path, "/") != "api/merchants" {
		t.Fatalf("expected api/merchants, received %v", path)
	}
}

func TestNormalizeLeavesAnUnresolvablePathEmpty(t *testing.T) {
	collection := mustParseCollection(t, `{"item":[{"name":"Hollow","request":{"url":{"raw":""}}}]}`)

	Normalize(collection, "atlas")

	if len(collection.Item[0].Request.URL.Path) != 0 {
		t.Fatalf("expected an empty path, received %v", collection.Item[0].Request.URL.Path)
	}
}
