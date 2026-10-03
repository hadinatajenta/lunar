package infrastructure

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"lunar/backend/internal/agent/domain"
)

var (
	dsmlBlockPattern  = regexp.MustCompile(`(?s)<[｜|]{1,2}DSML[｜|]{1,2}\s*(?:calls|tool_calls)>(.*?)</[｜|]{1,2}DSML[｜|]{1,2}\s*(?:calls|tool_calls)>`)
	dsmlInvokePattern = regexp.MustCompile(`(?s)<[｜|]{1,2}DSML[｜|]{1,2}\s*invoke\s+name="([^"]+)">(.*?)</[｜|]{1,2}DSML[｜|]{1,2}\s*invoke>`)
	dsmlParamPattern  = regexp.MustCompile(`(?s)<[｜|]{1,2}DSML[｜|]{1,2}\s*parameter\s+name="([^"]+)"(?:\s+string="([^"]*)")?\s*>(.*?)</[｜|]{1,2}DSML[｜|]{1,2}\s*parameter>`)
)

func ContainsDSML(text string) bool {
	return strings.Contains(text, "DSML") && (strings.Contains(text, "invoke") || strings.Contains(text, "calls"))
}

func ParseDSMLToolCalls(content string) ([]domain.ToolCall, string) {
	if !ContainsDSML(content) {
		return nil, content
	}

	var toolCalls []domain.ToolCall
	callIndex := 0

	blockMatches := dsmlBlockPattern.FindAllStringSubmatchIndex(content, -1)
	if len(blockMatches) == 0 {
		invokes := dsmlInvokePattern.FindAllStringSubmatch(content, -1)
		for _, invoke := range invokes {
			toolCalls = append(toolCalls, buildDSMLToolCall(invoke[1], invoke[2], callIndex))
			callIndex++
		}
		return toolCalls, strings.TrimSpace(dsmlInvokePattern.ReplaceAllString(content, ""))
	}

	for _, blockMatch := range blockMatches {
		blockBody := content[blockMatch[2]:blockMatch[3]]
		invokes := dsmlInvokePattern.FindAllStringSubmatch(blockBody, -1)
		for _, invoke := range invokes {
			toolCalls = append(toolCalls, buildDSMLToolCall(invoke[1], invoke[2], callIndex))
			callIndex++
		}
	}

	return toolCalls, strings.TrimSpace(dsmlBlockPattern.ReplaceAllString(content, ""))
}

func buildDSMLToolCall(functionName string, paramsBody string, callIndex int) domain.ToolCall {
	argumentValues := make(map[string]interface{})
	params := dsmlParamPattern.FindAllStringSubmatch(paramsBody, -1)
	for _, param := range params {
		paramName := param[1]
		stringAttribute := strings.ToLower(param[2])
		rawValue := strings.TrimSpace(param[3])

		if stringAttribute != "false" {
			argumentValues[paramName] = rawValue
			continue
		}
		argumentValues[paramName] = parseDSMLScalar(rawValue)
	}

	encodedArguments, err := json.Marshal(argumentValues)
	if err != nil {
		encodedArguments = []byte("{}")
	}

	return domain.ToolCall{
		ID:   dsmlCallID(callIndex),
		Type: defaultCallType,
		Function: domain.ToolCallFunction{
			Name:      functionName,
			Arguments: string(encodedArguments),
		},
	}
}

func parseDSMLScalar(rawValue string) interface{} {
	switch rawValue {
	case "true":
		return true
	case "false":
		return false
	}
	if integerValue, err := strconv.Atoi(rawValue); err == nil {
		return integerValue
	}
	if floatValue, err := strconv.ParseFloat(rawValue, 64); err == nil {
		return floatValue
	}
	var jsonValue interface{}
	if err := json.Unmarshal([]byte(rawValue), &jsonValue); err == nil {
		return jsonValue
	}
	return rawValue
}

func dsmlCallID(callIndex int) string {
	var suffix [4]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return fmt.Sprintf("call_dsml_%d", callIndex)
	}
	return fmt.Sprintf("call_dsml_%s_%d", hex.EncodeToString(suffix[:]), callIndex)
}
