package infrastructure

import (
	"strings"
	"testing"
)

func TestConfluenceStorageToHTML_RendersCodeMacro(t *testing.T) {
	storage := `<p>Intro</p><ac:structured-macro ac:name="code" ac:schema-version="1"><ac:parameter ac:name="language">go</ac:parameter><ac:plain-text-body><![CDATA[if a < b && c > d]]></ac:plain-text-body></ac:structured-macro>`

	result := confluenceStorageToHTML(storage, "")

	if !strings.Contains(result, "<pre><code>") {
		t.Errorf("expected code macro to render as pre/code, got %q", result)
	}
	if strings.Contains(result, "<ac:") || strings.Contains(result, "<ri:") {
		t.Errorf("expected no namespaced tags to survive, got %q", result)
	}
	if !strings.Contains(result, "if a &lt; b &amp;&amp; c &gt; d") {
		t.Errorf("expected code content to be HTML-escaped, got %q", result)
	}
	if strings.Contains(result, "language") {
		t.Errorf("expected ac:parameter payload to be dropped, got %q", result)
	}
}

func TestConfluenceStorageToHTML_RendersCalloutMacros(t *testing.T) {
	storage := `<ac:structured-macro ac:name="info"><ac:rich-text-body><p>Heads up</p></ac:rich-text-body></ac:structured-macro>`

	result := confluenceStorageToHTML(storage, "")

	if !strings.Contains(result, `<blockquote class="callout callout-info">`) {
		t.Errorf("expected info macro to render as callout blockquote, got %q", result)
	}
	if !strings.Contains(result, "<p>Heads up</p>") {
		t.Errorf("expected callout body to be preserved, got %q", result)
	}
}

func TestConfluenceStorageToHTML_RendersAttachmentImage(t *testing.T) {
	storage := `<ac:image ac:alt=""><ri:attachment ri:filename="diagram screen.png" /></ac:image>`

	result := confluenceStorageToHTML(storage, "https://confluence.example.com/download/attachments/77")

	if !strings.Contains(result, `src="https://confluence.example.com/download/attachments/77/diagram screen.png"`) {
		t.Errorf("expected image src built from attachment base, got %q", result)
	}
	if !strings.Contains(result, `alt="diagram screen.png"`) {
		t.Errorf("expected filename to become alt text, got %q", result)
	}
}

func TestConfluenceStorageToHTML_DropsImageWithoutAttachmentBase(t *testing.T) {
	storage := `<ac:image><ri:attachment ri:filename="x.png" /></ac:image>`

	result := confluenceStorageToHTML(storage, "")

	if result != "" {
		t.Errorf("expected image to be dropped when no attachment base is known, got %q", result)
	}
}

func TestConfluenceStorageToHTML_SanitizesAttachmentFilename(t *testing.T) {
	storage := `<ac:image><ri:attachment ri:filename="a&quot; onerror=&quot;alert(1)" /></ac:image>`

	result := confluenceStorageToHTML(storage, "https://confluence.example.com/att")

	if strings.Contains(result, "onerror=") {
		t.Errorf("expected filename to not inject an event handler attribute, got %q", result)
	}
	if strings.Contains(result, `"onerror`) || strings.Count(result, "<") != 1 {
		t.Errorf("expected quotes and angle brackets stripped from filename, got %q", result)
	}
}

func TestConfluenceStorageToHTML_RendersPageLinkAsTitle(t *testing.T) {
	storage := `<ac:link><ri:page ri:content-title="SOP-301 Database Migration" /><ac:plain-text-body><![CDATA[See this]]></ac:plain-text-body></ac:link>`

	result := confluenceStorageToHTML(storage, "")

	if !strings.Contains(result, "<strong>See this</strong>") {
		t.Errorf("expected link body text to be used, got %q", result)
	}
}

func TestConfluenceStorageToHTML_RendersPageLinkWithoutBodyAsTitle(t *testing.T) {
	storage := `<ac:link><ri:page ri:content-title="SOP-301 Database Migration" /></ac:link>`

	result := confluenceStorageToHTML(storage, "")

	if !strings.Contains(result, "<strong>SOP-301 Database Migration</strong>") {
		t.Errorf("expected content title to be used when no body present, got %q", result)
	}
}

func TestConfluenceStorageToHTML_RendersTaskList(t *testing.T) {
	storage := `<ac:task-list><ac:task><ac:task-status>complete</ac:task-status><ac:task-body><![CDATA[Deploy to staging]]></ac:task-body></ac:task><ac:task><ac:task-status>TODO</ac:task-status><ac:task-body><![CDATA[Run smoke tests]]></ac:task-body></ac:task></ac:task-list>`

	result := confluenceStorageToHTML(storage, "")

	if !strings.Contains(result, `<ul class="task-list">`) {
		t.Errorf("expected task list to render as ul, got %q", result)
	}
	if !strings.Contains(result, "Deploy to staging") || !strings.Contains(result, "Run smoke tests") {
		t.Errorf("expected both task bodies to render, got %q", result)
	}
	if !strings.Contains(result, "&#9745;") {
		t.Errorf("expected completed task to render a checked marker, got %q", result)
	}
	if strings.Contains(result, "ac:task") {
		t.Errorf("expected task tags removed, got %q", result)
	}
}

func TestConfluenceStorageToHTML_StripsUnknownNamespacedTags(t *testing.T) {
	storage := `<ac:layout><ac:layout-section><ac:layout-cell><p>Kept</p><ri:page ri:content-title="Other" /><ac:emoticon ac:name="smile"/></ac:layout-cell></ac:layout-section></ac:layout>`

	result := confluenceStorageToHTML(storage, "")

	if !strings.Contains(result, "<p>Kept</p>") {
		t.Errorf("expected plain paragraph inside layout to survive, got %q", result)
	}
	if strings.Contains(result, "<ac:") || strings.Contains(result, "<ri:") {
		t.Errorf("expected all namespaced tags stripped, got %q", result)
	}
}

func TestConfluenceStorageToHTML_EscapesStrayCDATA(t *testing.T) {
	storage := `<ac:some-macro><ac:body><![CDATA[<script>alert(1)</script>]]></ac:body></ac:some-macro>`

	result := confluenceStorageToHTML(storage, "")

	if strings.Contains(result, "<script>") {
		t.Errorf("expected script tag to be escaped, got %q", result)
	}
	if !strings.Contains(result, "&lt;script&gt;") {
		t.Errorf("expected escaped script text, got %q", result)
	}
}

func TestConfluenceStorageToHTML_LeavesStandardHTMLUntouched(t *testing.T) {
	storage := `<h2>Heading</h2><table><tbody><tr><td>Cell</td></tr></tbody></table><ul><li>Item</li></ul>`

	result := confluenceStorageToHTML(storage, "")

	for _, fragment := range []string{"<h2>Heading</h2>", "<table>", "<td>Cell</td>", "<li>Item</li>"} {
		if !strings.Contains(result, fragment) {
			t.Errorf("expected %q to survive, got %q", fragment, result)
		}
	}
}
