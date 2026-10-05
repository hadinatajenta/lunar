package application

import (
	"fmt"
	"strings"

	"lunar/backend/internal/postman/domain"
)

type requestEntry struct {
	folder string
	item   domain.CollectionItem
}

func Validate(collection *domain.Collection) error {
	requests := collectRequests(collection.Item, "")
	if len(requests) == 0 {
		return fmt.Errorf("%w: no endpoints found in the collection", domain.ErrCollectionInvalid)
	}
	for _, entry := range requests {
		if err := validateRequest(entry); err != nil {
			return err
		}
	}
	return nil
}

func collectRequests(items []domain.CollectionItem, folder string) []requestEntry {
	requests := make([]requestEntry, 0, len(items))
	for _, item := range items {
		if item.Request != nil {
			requests = append(requests, requestEntry{folder: folder, item: item})
			continue
		}
		requests = append(requests, collectRequests(item.Item, item.Name)...)
	}
	return requests
}

func validateRequest(entry requestEntry) error {
	location := describeLocation(entry.folder, entry.item.Name)
	if strings.TrimSpace(entry.item.Name) == "" {
		return fmt.Errorf("%w: a request in %s has no name", domain.ErrCollectionInvalid, location)
	}

	request := entry.item.Request
	if strings.TrimSpace(request.Method) == "" {
		return fmt.Errorf("%w: request %s has no method", domain.ErrCollectionInvalid, location)
	}
	if request.URL == nil || len(request.URL.Path) == 0 {
		return fmt.Errorf("%w: request %s has no url path", domain.ErrCollectionInvalid, location)
	}
	if request.Header == nil {
		return fmt.Errorf("%w: request %s has no header array", domain.ErrCollectionInvalid, location)
	}
	for _, parameter := range request.URL.Query {
		if strings.TrimSpace(parameter.Key) == "" {
			return fmt.Errorf("%w: request %s has a query parameter without a key", domain.ErrCollectionInvalid, location)
		}
	}
	if request.Body == nil {
		return nil
	}
	for _, field := range request.Body.URLEncoded {
		if strings.TrimSpace(field.Key) == "" {
			return fmt.Errorf("%w: request %s has a body field without a key", domain.ErrCollectionInvalid, location)
		}
	}
	for _, field := range request.Body.FormData {
		if strings.TrimSpace(field.Key) == "" {
			return fmt.Errorf("%w: request %s has a body field without a key", domain.ErrCollectionInvalid, location)
		}
	}
	return nil
}

func describeLocation(folder string, name string) string {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		trimmedName = "unnamed"
	}
	if strings.TrimSpace(folder) == "" {
		return fmt.Sprintf("%q", trimmedName)
	}
	return fmt.Sprintf("%q in folder %q", trimmedName, strings.TrimSpace(folder))
}

func CountRequests(collection *domain.Collection) int {
	return len(collectRequests(collection.Item, ""))
}

func CountFolders(collection *domain.Collection) int {
	return countFolders(collection.Item)
}

func countFolders(items []domain.CollectionItem) int {
	folders := 0
	for _, item := range items {
		if item.Request != nil {
			continue
		}
		if len(collectRequests(item.Item, "")) > 0 {
			folders++
		}
		folders += countFolders(item.Item)
	}
	return folders
}
