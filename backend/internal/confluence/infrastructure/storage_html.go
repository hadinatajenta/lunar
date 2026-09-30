package infrastructure

import (
	"html"
	"regexp"
	"strings"
)

var (
	storageCodeMacroPattern      = regexp.MustCompile(`(?s)<ac:structured-macro[^>]*ac:name="(?:code|code-block|pre|noformat)"[^>]*>.*?</ac:structured-macro>`)
	storagePlainTextBodyPattern  = regexp.MustCompile(`(?s)<ac:plain-text-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-body>`)
	storageCalloutMacroPattern   = regexp.MustCompile(`(?s)<ac:structured-macro[^>]*ac:name="(info|note|warning|tip)"[^>]*>(.*?)</ac:structured-macro>`)
	storageRichTextBodyPattern   = regexp.MustCompile(`(?s)<ac:rich-text-body>(.*?)</ac:rich-text-body>`)
	storageImagePattern          = regexp.MustCompile(`(?s)<ac:image[^>]*>\s*<ri:attachment[^>]*ri:filename="([^"]*)"[^>]*/?>.*?</ac:image>`)
	storagePageLinkPattern       = regexp.MustCompile(`(?s)<ac:link>\s*<ri:page[^>]*ri:content-title="([^"]*)"[^>]*/?>\s*(?:<ac:plain-text-body><!\[CDATA\[(.*?)\]\]></ac:plain-text-body>)?\s*</ac:link>`)
	storageTaskListPattern       = regexp.MustCompile(`(?s)<ac:task-list>(.*?)</ac:task-list>`)
	storageTaskPattern           = regexp.MustCompile(`(?s)<ac:task>(.*?)</ac:task>`)
	storageTaskStatusPattern     = regexp.MustCompile(`(?s)<ac:task-status>(.*?)</ac:task-status>`)
	storageTaskBodyPattern       = regexp.MustCompile(`(?s)<ac:task-body><!\[CDATA\[(.*?)\]\]></ac:task-body>`)
	storageForeignTagPattern     = regexp.MustCompile(`</?(?:ac|ri)(?::[a-z0-9-]+)+[^>]*>`)
	storageCDATASectionPattern   = regexp.MustCompile(`(?s)<!\[CDATA\[(.*?)\]\]>`)
	storageAttachmentNamePattern = regexp.MustCompile(`[^A-Za-z0-9._ -]`)
)

func confluenceStorageToHTML(storage string, attachmentBase string) string {
	result := storage

	result = storageCodeMacroPattern.ReplaceAllStringFunc(result, func(macro string) string {
		body := storagePlainTextBodyPattern.FindStringSubmatch(macro)
		if body == nil {
			return ""
		}
		return "<pre><code>" + html.EscapeString(body[1]) + "</code></pre>"
	})

	result = storageCalloutMacroPattern.ReplaceAllStringFunc(result, func(macro string) string {
		parts := storageCalloutMacroPattern.FindStringSubmatch(macro)
		inner := storageRichTextBodyPattern.FindStringSubmatch(parts[2])
		body := ""
		if inner != nil {
			body = inner[1]
		}
		return `<blockquote class="callout callout-` + parts[1] + `">` + body + `</blockquote>`
	})

	result = storageImagePattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := storageImagePattern.FindStringSubmatch(match)
		filename := storageAttachmentNamePattern.ReplaceAllString(parts[1], "")
		filename = strings.TrimSpace(filename)
		if filename == "" || attachmentBase == "" {
			return ""
		}
		return `<img src="` + html.EscapeString(attachmentBase+"/"+filename) +
			`" alt="` + html.EscapeString(filename) + `">`
	})

	result = storagePageLinkPattern.ReplaceAllStringFunc(result, func(match string) string {
		parts := storagePageLinkPattern.FindStringSubmatch(match)
		title := parts[1]
		text := title
		if parts[2] != "" {
			text = parts[2]
		}
		return "<strong>" + html.EscapeString(text) + "</strong>"
	})

	result = storageTaskListPattern.ReplaceAllStringFunc(result, func(match string) string {
		inner := storageTaskListPattern.FindStringSubmatch(match)[1]
		inner = storageTaskPattern.ReplaceAllStringFunc(inner, func(task string) string {
			taskParts := storageTaskPattern.FindStringSubmatch(task)
			status := storageTaskStatusPattern.FindStringSubmatch(taskParts[1])
			taskBody := storageTaskBodyPattern.FindStringSubmatch(taskParts[1])
			bodyText := ""
			if taskBody != nil {
				bodyText = html.EscapeString(taskBody[1])
			}
			marker := "&#9744; "
			if status != nil && strings.EqualFold(strings.TrimSpace(status[1]), "complete") {
				marker = "&#9745; "
			}
			return "<li>" + marker + bodyText + "</li>"
		})
		return `<ul class="task-list">` + inner + `</ul>`
	})

	result = storageForeignTagPattern.ReplaceAllString(result, "")
	result = storageCDATASectionPattern.ReplaceAllStringFunc(result, func(match string) string {
		return html.EscapeString(storageCDATASectionPattern.FindStringSubmatch(match)[1])
	})
	return result
}
