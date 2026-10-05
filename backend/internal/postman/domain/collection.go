package domain

import (
	"encoding/json"
	"strings"
)

const CollectionSchemaURL = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

type Collection struct {
	Info                    CollectionInfo       `json:"info"`
	Item                    []CollectionItem     `json:"item"`
	Variable                []CollectionVariable `json:"variable"`
	Auth                    json.RawMessage      `json:"auth,omitempty"`
	Event                   []CollectionEvent    `json:"event,omitempty"`
	ProtocolProfileBehavior json.RawMessage      `json:"protocolProfileBehavior,omitempty"`
}

type CollectionInfo struct {
	Name        string          `json:"name"`
	Schema      string          `json:"schema"`
	ID          string          `json:"_postman_id,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
	Version     json.RawMessage `json:"version,omitempty"`
}

type CollectionItem struct {
	Name                    string               `json:"name"`
	Description             json.RawMessage      `json:"description,omitempty"`
	Item                    []CollectionItem     `json:"item,omitempty"`
	Request                 *CollectionRequest   `json:"request,omitempty"`
	Response                json.RawMessage      `json:"response,omitempty"`
	Event                   []CollectionEvent    `json:"event,omitempty"`
	Auth                    json.RawMessage      `json:"auth,omitempty"`
	Variable                []CollectionVariable `json:"variable,omitempty"`
	ProtocolProfileBehavior json.RawMessage      `json:"protocolProfileBehavior,omitempty"`
}

type CollectionRequest struct {
	Method      string          `json:"method"`
	Header      []RequestHeader `json:"header"`
	Body        *RequestBody    `json:"body,omitempty"`
	URL         *RequestURL     `json:"url,omitempty"`
	Auth        json.RawMessage `json:"auth,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
	Proxy       json.RawMessage `json:"proxy,omitempty"`
	Certificate json.RawMessage `json:"certificate,omitempty"`
}

type RequestHeader struct {
	Key         string          `json:"key"`
	Value       string          `json:"value"`
	Type        string          `json:"type,omitempty"`
	Disabled    bool            `json:"disabled,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
}

type RequestURL struct {
	Raw      string           `json:"raw,omitempty"`
	Protocol string           `json:"protocol,omitempty"`
	Host     []string         `json:"host,omitempty"`
	Port     string           `json:"port,omitempty"`
	Path     PathSegments     `json:"path,omitempty"`
	Query    []QueryParameter `json:"query,omitempty"`
	Hash     string           `json:"hash,omitempty"`
	Variable []URLVariable    `json:"variable,omitempty"`
}

type PathSegments []string

func (p *PathSegments) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "null" || trimmed == "\"\"" {
		*p = nil
		return nil
	}

	if strings.HasPrefix(trimmed, "[") {
		var segments []string
		if err := json.Unmarshal(data, &segments); err != nil {
			return err
		}
		*p = segments
		return nil
	}

	var single string
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*p = SplitPathSegments(single)
	return nil
}

func SplitPathSegments(value string) []string {
	segments := make([]string, 0, 4)
	for _, segment := range strings.Split(value, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return nil
	}
	return segments
}

type QueryParameter struct {
	Key         string          `json:"key"`
	Value       string          `json:"value"`
	Disabled    bool            `json:"disabled,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
}

type URLVariable struct {
	Key         string          `json:"key"`
	Value       string          `json:"value"`
	Type        string          `json:"type,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
}

type RequestBody struct {
	Mode       string          `json:"mode,omitempty"`
	Raw        string          `json:"raw,omitempty"`
	URLEncoded []BodyField     `json:"urlencoded,omitempty"`
	FormData   []BodyField     `json:"formdata,omitempty"`
	File       json.RawMessage `json:"file,omitempty"`
	GraphQL    json.RawMessage `json:"graphql,omitempty"`
	Options    json.RawMessage `json:"options,omitempty"`
	Disabled   bool            `json:"disabled,omitempty"`
}

type BodyField struct {
	Key         string          `json:"key"`
	Value       string          `json:"value,omitempty"`
	Type        string          `json:"type,omitempty"`
	Src         json.RawMessage `json:"src,omitempty"`
	Disabled    bool            `json:"disabled,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
	ContentType string          `json:"contentType,omitempty"`
}

type CollectionVariable struct {
	Key         string          `json:"key"`
	Value       string          `json:"value"`
	Type        string          `json:"type,omitempty"`
	Disabled    bool            `json:"disabled,omitempty"`
	Description json.RawMessage `json:"description,omitempty"`
}

type CollectionEvent struct {
	Listen   string       `json:"listen"`
	Script   *EventScript `json:"script,omitempty"`
	Disabled bool         `json:"disabled,omitempty"`
}

type EventScript struct {
	ID   string          `json:"id,omitempty"`
	Type string          `json:"type,omitempty"`
	Name string          `json:"name,omitempty"`
	Exec []string        `json:"exec,omitempty"`
	Src  json.RawMessage `json:"src,omitempty"`
}
