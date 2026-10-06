package domain

import (
	"encoding/json"
	"strconv"
	"strings"
)

type JiraSprintValues []string

type SprintReference struct {
	Name    string
	BoardID int
}

var sprintFieldKeys = []string{
	"id=",
	"rapidViewId=",
	"state=",
	"name=",
	"startDate=",
	"endDate=",
	"completeDate=",
	"activatedDate=",
	"sequence=",
	"goal=",
	"synced=",
	"autoStartStop=",
	"incompleteIssuesDestinationId=",
}

func (values *JiraSprintValues) UnmarshalJSON(payload []byte) error {
	trimmedPayload := strings.TrimSpace(string(payload))
	if trimmedPayload == "" || trimmedPayload == "null" {
		*values = nil
		return nil
	}

	var singleValue string
	if err := json.Unmarshal(payload, &singleValue); err == nil {
		if strings.TrimSpace(singleValue) == "" {
			*values = nil
			return nil
		}
		*values = JiraSprintValues{singleValue}
		return nil
	}

	var multipleValues []string
	if err := json.Unmarshal(payload, &multipleValues); err == nil {
		if len(multipleValues) == 0 {
			*values = nil
			return nil
		}
		*values = JiraSprintValues(multipleValues)
		return nil
	}

	*values = nil
	return nil
}

func ParseSprintReference(rawValue string) SprintReference {
	return SprintReference{
		Name:    extractSprintField(rawValue, "name="),
		BoardID: parseSprintBoardID(extractSprintField(rawValue, "rapidViewId=")),
	}
}

func extractSprintField(rawValue string, fieldKey string) string {
	searchFrom := 0
	for {
		relativeIndex := strings.Index(rawValue[searchFrom:], fieldKey)
		if relativeIndex == -1 {
			return ""
		}
		fieldIndex := searchFrom + relativeIndex
		if isSprintFieldBoundary(rawValue, fieldIndex) {
			return trimSprintFieldValue(rawValue[fieldIndex+len(fieldKey):])
		}
		searchFrom = fieldIndex + 1
	}
}

func isSprintFieldBoundary(rawValue string, fieldIndex int) bool {
	if fieldIndex == 0 {
		return true
	}
	previous := rawValue[fieldIndex-1]
	return previous == '[' || previous == ','
}

func trimSprintFieldValue(remainder string) string {
	if closingIndex := strings.IndexByte(remainder, ']'); closingIndex != -1 {
		remainder = remainder[:closingIndex]
	}
	terminatorIndex := len(remainder)
	for _, fieldKey := range sprintFieldKeys {
		if keyIndex := strings.Index(remainder, ","+fieldKey); keyIndex != -1 && keyIndex < terminatorIndex {
			terminatorIndex = keyIndex
		}
	}
	return strings.TrimSpace(remainder[:terminatorIndex])
}

func parseSprintBoardID(boardValue string) int {
	boardID, err := strconv.Atoi(strings.TrimSpace(boardValue))
	if err != nil {
		return 0
	}
	return boardID
}
