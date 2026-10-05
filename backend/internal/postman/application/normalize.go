package application

import (
	"encoding/json"
	"net/http"
	"strings"

	"lunar/backend/internal/postman/domain"
)

const baseURLVariable = "{{baseUrl}}"

const emptyJSONArray = "[]"

func Normalize(collection *domain.Collection, service string) {
	normalizeInfo(&collection.Info, service)
	if collection.Item == nil {
		collection.Item = []domain.CollectionItem{}
	}
	if collection.Variable == nil {
		collection.Variable = []domain.CollectionVariable{}
	}
	for index := range collection.Item {
		normalizeItem(&collection.Item[index])
	}
}

func normalizeInfo(info *domain.CollectionInfo, service string) {
	if strings.TrimSpace(info.Name) == "" {
		info.Name = service + " API"
	}
	if strings.TrimSpace(info.Schema) == "" {
		info.Schema = domain.CollectionSchemaURL
	}
}

func normalizeItem(item *domain.CollectionItem) {
	for index := range item.Item {
		normalizeItem(&item.Item[index])
	}
	if item.Request == nil {
		return
	}
	normalizeRequest(item.Request)
	if len(item.Response) == 0 {
		item.Response = json.RawMessage(emptyJSONArray)
	}
}

func normalizeRequest(request *domain.CollectionRequest) {
	if strings.TrimSpace(request.Method) == "" {
		request.Method = http.MethodGet
	}
	if request.Header == nil {
		request.Header = []domain.RequestHeader{}
	}
	if request.URL == nil {
		request.URL = &domain.RequestURL{}
	}
	normalizeURL(request.URL)
}

func normalizeURL(url *domain.RequestURL) {
	if len(url.Host) == 0 {
		url.Host = []string{baseURLVariable}
	}
	if len(url.Path) == 0 {
		url.Path = domain.PathSegments(derivePathFromRaw(url.Raw))
	}
}

func derivePathFromRaw(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	if separator := strings.Index(trimmed, "://"); separator >= 0 {
		remainder := trimmed[separator+3:]
		if slash := strings.Index(remainder, "/"); slash >= 0 {
			remainder = remainder[slash:]
		} else {
			remainder = ""
		}
		trimmed = remainder
	} else if strings.HasPrefix(trimmed, "{{") {
		if closing := strings.Index(trimmed, "}}"); closing >= 0 {
			trimmed = trimmed[closing+2:]
		}
	}

	if boundary := strings.IndexAny(trimmed, "?#"); boundary >= 0 {
		trimmed = trimmed[:boundary]
	}
	return domain.SplitPathSegments(trimmed)
}
