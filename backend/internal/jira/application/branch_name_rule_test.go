package application

import "testing"

func TestFallbackCoreMatchesTheMeasuredContractNames(t *testing.T) {
	tests := []struct {
		name     string
		summary  string
		wantCore string
	}{
		{
			name:     "bulk MDR QRIS Statis",
			summary:  "Sebagai Maker & Approval di New MMS, I want to memiliki menu dan form terpisah untuk mengajukan program bulk MDR QRIS Statis",
			wantCore: "Menu-Form-Terpisah-Program-Bulk",
		},
		{
			name:     "view and change the managing PN form",
			summary:  "Sebagai Maker, I want to melihat & mengubah PN pengelola pada form pengajuan",
			wantCore: "Melihat-Mengubah-PN-Pengelola-Form",
		},
		{
			name:     "summary without an intent clause",
			summary:  "Enhancement Manage User",
			wantCore: "Enhancement-Manage-User",
		},
		{
			name:     "keeps an all caps word and a number",
			summary:  "QRIS Open API max 30",
			wantCore: "QRIS-Open-API-Max-30",
		},
		{
			name:     "caps the core at five words",
			summary:  "satu dua tiga empat lima enam tujuh",
			wantCore: "Satu-Dua-Tiga-Empat-Lima",
		},
		{
			name:     "drops the Indonesian filler the contract names",
			summary:  "sebagai di dan untuk yang dengan memiliki mengajukan pada dapat admin",
			wantCore: "Admin",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			core, evidence := fallbackCore(testCase.summary)
			if core != testCase.wantCore {
				t.Fatalf("core = %q, want %q", core, testCase.wantCore)
			}
			if evidence == "" {
				t.Fatal("the fallback must name the phrase it used as evidence")
			}
		})
	}
}

func TestFallbackEvidenceIsTheIntentClause(t *testing.T) {
	_, evidence := fallbackCore("Sebagai Maker, I Want To melihat daftar user")
	if evidence != "melihat daftar user" {
		t.Fatalf("evidence = %q, want the clause after the intent marker", evidence)
	}
}

func TestFallbackEvidenceIsTheWholeSummaryWithoutTheMarker(t *testing.T) {
	summary := "Enhancement Manage User"
	_, evidence := fallbackCore(summary)
	if evidence != summary {
		t.Fatalf("evidence = %q, want %q", evidence, summary)
	}
}

func TestFallbackCoreIsEmptyForASummaryWithoutWords(t *testing.T) {
	core, evidence := fallbackCore("!!! ???")
	if core != "" {
		t.Fatalf("core = %q, want an empty core", core)
	}
	if evidence != "!!! ???" {
		t.Fatalf("evidence = %q, want the summary verbatim", evidence)
	}
}

func TestNameWordsSplitOnAnythingThatIsNotALetterOrADigit(t *testing.T) {
	words := nameWords("halo, dunia_baru 2x (PN)")
	want := []string{"Halo", "Dunia", "Baru", "2x", "PN"}
	if len(words) != len(want) {
		t.Fatalf("words = %v, want %v", words, want)
	}
	for index := range want {
		if words[index] != want[index] {
			t.Fatalf("words = %v, want %v", words, want)
		}
	}
}
