package infrastructure

import (
	"fmt"
	"strings"
	"testing"

	codeindexdomain "lunar/backend/internal/codeindex/domain"
)

const phpFixture = `<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;

class AddOutletController extends Controller
{
    public function store(Request $request)
    {
        $status = [];
        $status['app_jenis'] = 2;

        if ($request->has('outlet_id')) {
            $message = "Outlet ID: " . $request->outlet_id;
            return response()->json(['status' => 'OK', 'app' => $status]);
        }

        return response()->json(['error' => 'Not found'], 404);
    }

    public function index()
    {
        Route::get('/api/outlets', 'OutletController@index');
    }
}
`

func TestExtractPHPReportsStoreFunctionRange(t *testing.T) {
	sourcePath, content := writeFixtureFile(t, "AddOutletController.php", phpFixture)
	chunks, _ := extractPHP(content, "app/Http/Controllers/AddOutletController.php", "merchant-service-laravel")
	assertChunkRangesMatchSource(t, sourcePath, chunks)

	storeChunk := findChunkContaining(chunks, "public function store")
	if storeChunk == nil {
		t.Fatal("expected a function chunk for store()")
	}
	if storeChunk.ChunkType != string(codeindexdomain.ChunkTypeCodeBlock) {
		t.Errorf("expected code_block chunk type, got %q", storeChunk.ChunkType)
	}
	if storeChunk.StartLine != 9 || storeChunk.EndLine != 20 {
		t.Errorf("expected store() on lines 9-20, got %d-%d", storeChunk.StartLine, storeChunk.EndLine)
	}
	if !strings.Contains(storeChunk.RawContent, "$status['app_jenis'] = 2;") {
		t.Errorf("expected store() chunk to contain the field assignment, got %q", storeChunk.RawContent)
	}
}

func TestExtractPHPReportsFieldAssignmentAndRouteRanges(t *testing.T) {
	sourcePath, content := writeFixtureFile(t, "AddOutletController.php", phpFixture)
	chunks, _ := extractPHP(content, "app/Http/Controllers/AddOutletController.php", "merchant-service-laravel")
	assertChunkRangesMatchSource(t, sourcePath, chunks)

	fieldChunk := findChunkWithTrimmedContent(chunks, "$status['app_jenis'] = 2;")
	if fieldChunk == nil || fieldChunk.StartLine != 12 || fieldChunk.EndLine != 12 {
		t.Errorf("expected field assignment on line 12, got %+v", fieldChunk)
	}

	routeChunk := findChunkContaining(chunks, "Route::get('/api/outlets'")
	if routeChunk == nil {
		t.Fatal("expected a route chunk for /api/outlets")
	}
	if routeChunk.ChunkType != string(codeindexdomain.ChunkTypeRoute) {
		t.Errorf("expected route chunk type, got %q", routeChunk.ChunkType)
	}
	if routeChunk.StartLine != 24 || routeChunk.EndLine != 24 {
		t.Errorf("expected route on line 24, got %d-%d", routeChunk.StartLine, routeChunk.EndLine)
	}
}

func TestExtractPHPSplitsLongFunctionIntoRangedBlocks(t *testing.T) {
	var fixtureBuilder strings.Builder
	fixtureBuilder.WriteString("<?php\n\nclass LongController\n{\n    public function longProcess()\n    {\n")
	for step := 1; step <= 300; step++ {
		fmt.Fprintf(&fixtureBuilder, "        $step%d = %d;\n", step, step)
	}
	fixtureBuilder.WriteString("        return true;\n    }\n}\n")

	sourcePath, content := writeFixtureFile(t, "LongController.php", fixtureBuilder.String())
	chunks, _ := extractPHP(content, "app/LongController.php", "merchant-service-laravel")
	assertChunkRangesMatchSource(t, sourcePath, chunks)

	if len(chunks) != 4 {
		t.Fatalf("expected 4 split blocks for a 304 line function, got %d", len(chunks))
	}
	if chunks[0].StartLine != 5 || chunks[0].EndLine != 104 {
		t.Errorf("expected first block on lines 5-104, got %d-%d", chunks[0].StartLine, chunks[0].EndLine)
	}
	if chunks[1].StartLine != 85 {
		t.Errorf("expected second block to overlap at line 85, got %d", chunks[1].StartLine)
	}
	if !strings.Contains(chunks[3].RawContent, "$step300 = 300;") {
		t.Errorf("expected last block to hold the final assignment, got %q", chunks[3].RawContent)
	}
}
