package fresh

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockRegistry returns an httptest.Server that serves npm-style metadata for
// a fixed set of packages. Callers must call srv.Close() when done.
func mockRegistry(t *testing.T, packages map[string]npmMeta) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		meta, ok := packages[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(meta)
	}))
}

// isoTime formats t as an RFC3339 string suitable for npm registry responses.
func isoTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

// TestBuildResult covers timestamp parsing and risk classification.
func TestBuildResult(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name      string
		rawTime   string
		ageDays   int
		wantRisk  Risk
		wantError bool
	}{
		{
			name:     "RFC3339 high risk",
			rawTime:  isoTime(now.Add(-1 * 24 * time.Hour)),
			ageDays:  7,
			wantRisk: RiskHigh,
		},
		{
			name:     "RFC3339 warn risk",
			rawTime:  isoTime(now.Add(-5 * 24 * time.Hour)),
			ageDays:  7,
			wantRisk: RiskWarn,
		},
		{
			name:     "RFC3339 OK",
			rawTime:  isoTime(now.Add(-30 * 24 * time.Hour)),
			ageDays:  7,
			wantRisk: RiskOK,
		},
		{
			name:     "millisecond format (npm fallback)",
			rawTime:  now.Add(-2 * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z"),
			ageDays:  7,
			wantRisk: RiskHigh,
		},
		{
			name:      "unparseable timestamp returns error",
			rawTime:   "not-a-date",
			ageDays:   7,
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := buildResult("pkg", "1.0.0", tc.rawTime, tc.ageDays)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Risk != tc.wantRisk {
				t.Errorf("risk = %v, want %v", res.Risk, tc.wantRisk)
			}
			if res.Package != "pkg" || res.Version != "1.0.0" {
				t.Errorf("got package=%q version=%q, want pkg/1.0.0", res.Package, res.Version)
			}
		})
	}
}

// TestParseLock covers package-lock.json v2/v3 parsing.
func TestParseLock(t *testing.T) {
	lock := `{
		"lockfileVersion": 3,
		"packages": {
			"": {},
			"node_modules/lodash": {"version": "4.17.21", "dev": false},
			"node_modules/typescript": {"version": "5.4.0", "dev": true},
			"node_modules/@types/node": {"version": "20.0.0", "dev": true}
		}
	}`

	path := filepath.Join(t.TempDir(), "package-lock.json")
	if err := os.WriteFile(path, []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("excludes dev deps by default", func(t *testing.T) {
		deps, err := parseLock(path, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(deps) != 1 || deps[0].name != "lodash" || deps[0].version != "4.17.21" {
			t.Errorf("unexpected deps: %+v", deps)
		}
	})

	t.Run("includes dev deps when requested", func(t *testing.T) {
		deps, err := parseLock(path, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(deps) != 3 {
			t.Errorf("expected 3 deps, got %d: %+v", len(deps), deps)
		}
	})

	t.Run("root entry (empty key) is skipped", func(t *testing.T) {
		deps, err := parseLock(path, true)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range deps {
			if d.name == "" {
				t.Error("empty-name dep (root entry) should be skipped")
			}
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "package-lock.json")
		os.WriteFile(bad, []byte("{bad json"), 0o644)
		_, err := parseLock(bad, false)
		if err == nil {
			t.Fatal("expected error for malformed JSON")
		}
	})
}

// TestParsePkg covers package.json parsing.
func TestParsePkg(t *testing.T) {
	pkg := `{
		"dependencies": {"axios": "^1.6.0", "lodash": "^4.0.0"},
		"devDependencies": {"typescript": "^5.0.0"}
	}`

	path := filepath.Join(t.TempDir(), "package.json")
	if err := os.WriteFile(path, []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("excludes dev deps by default", func(t *testing.T) {
		deps, err := parsePkg(path, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(deps) != 2 {
			t.Errorf("expected 2 deps, got %d", len(deps))
		}
		// Version should be empty (resolved from registry at fetch time)
		for _, d := range deps {
			if d.version != "" {
				t.Errorf("dep %q: expected empty version from package.json, got %q", d.name, d.version)
			}
		}
	})

	t.Run("includes dev deps when requested", func(t *testing.T) {
		deps, err := parsePkg(path, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(deps) != 3 {
			t.Errorf("expected 3 deps, got %d", len(deps))
		}
	})
}

// TestCheck_WithLockfile is an integration test that wires a mock registry
// through the full Check call using a package-lock.json.
func TestCheck_WithLockfile(t *testing.T) {
	now := time.Now().UTC()

	srv := mockRegistry(t, map[string]npmMeta{
		"lodash": {
			DistTags: map[string]string{"latest": "4.17.21"},
			Time:     map[string]string{"4.17.21": isoTime(now.Add(-365 * 24 * time.Hour))},
		},
		"new-pkg": {
			DistTags: map[string]string{"latest": "1.0.0"},
			Time:     map[string]string{"1.0.0": isoTime(now.Add(-1 * 24 * time.Hour))},
		},
	})
	defer srv.Close()

	lock := `{
		"lockfileVersion": 3,
		"packages": {
			"node_modules/lodash":  {"version": "4.17.21"},
			"node_modules/new-pkg": {"version": "1.0.0"}
		}
	}`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Check(dir, Options{AgeDays: 7, RegistryURL: srv.URL})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	byPkg := make(map[string]Result, len(results))
	for _, r := range results {
		byPkg[r.Package] = r
	}

	if byPkg["lodash"].Risk != RiskOK {
		t.Errorf("lodash: want RiskOK, got %v", byPkg["lodash"].Risk)
	}
	if byPkg["new-pkg"].Risk != RiskHigh {
		t.Errorf("new-pkg: want RiskHigh, got %v", byPkg["new-pkg"].Risk)
	}
}

// TestCheck_WithPackageJSON tests the fallback path (no lockfile, uses dist-tags.latest).
func TestCheck_WithPackageJSON(t *testing.T) {
	now := time.Now().UTC()

	srv := mockRegistry(t, map[string]npmMeta{
		"axios": {
			DistTags: map[string]string{"latest": "1.6.0"},
			Time:     map[string]string{"1.6.0": isoTime(now.Add(-5 * 24 * time.Hour))},
		},
	})
	defer srv.Close()

	pkg := `{"dependencies": {"axios": "^1.6.0"}}`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := Check(dir, Options{AgeDays: 7, RegistryURL: srv.URL})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Version != "1.6.0" {
		t.Errorf("version = %q, want 1.6.0", results[0].Version)
	}
	if results[0].Risk != RiskWarn {
		t.Errorf("risk = %v, want RiskWarn", results[0].Risk)
	}
}

// TestParseBunLock covers bun.lock JSONC parsing and name@version splitting.
func TestParseBunLock(t *testing.T) {
	// Realistic bun.lock snippet: keys are package names (no version), first
	// array element is canonical "name@version". Path-based keys are also
	// present for conflict resolution — same package, should be deduplicated.
	lock := `{
		"lockfileVersion": 1,
		"workspaces": {
			"": {
				"name": "my-app",
				"dependencies": { "lodash": "4.17.21", },
				"devDependencies": { "typescript": "5.4.0", },
			},
		},
		"packages": {
			"lodash": ["lodash@4.17.21", "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz", {}, "sha512-abc"],
			"@types/react": ["@types/react@19.2.14", "https://registry.npmjs.org/@types/react/-/react-19.2.14.tgz", {}, "sha512-xyz"],
			"typescript": ["typescript@5.4.0", "https://registry.npmjs.org/typescript/-/typescript-5.4.0.tgz", {}, "sha512-def"],
			"some-parent/@types/react": ["@types/react@19.2.14", "https://registry.npmjs.org/@types/react/-/react-19.2.14.tgz", {}, "sha512-xyz"],
		},
	}`

	path := filepath.Join(t.TempDir(), "bun.lock")
	if err := os.WriteFile(path, []byte(lock), 0o644); err != nil {
		t.Fatal(err)
	}

	deps, err := parseBunLock(path)
	if err != nil {
		t.Fatalf("parseBunLock: %v", err)
	}

	// 4 entries in packages section but @types/react appears twice (deduped to 3).
	if len(deps) != 3 {
		t.Fatalf("expected 3 deps (deduped), got %d: %+v", len(deps), deps)
	}

	byName := make(map[string]string, len(deps))
	for _, d := range deps {
		byName[d.name] = d.version
	}

	if byName["lodash"] != "4.17.21" {
		t.Errorf("lodash version = %q, want 4.17.21", byName["lodash"])
	}
	if byName["@types/react"] != "19.2.14" {
		t.Errorf("@types/react version = %q, want 19.2.14", byName["@types/react"])
	}
	if byName["typescript"] != "5.4.0" {
		t.Errorf("typescript version = %q, want 5.4.0", byName["typescript"])
	}
}

func TestSplitBunPkgKey(t *testing.T) {
	tests := []struct {
		key         string
		wantName    string
		wantVersion string
		wantOK      bool
	}{
		{"lodash@4.17.21", "lodash", "4.17.21", true},
		{"@types/react@19.2.14", "@types/react", "19.2.14", true},
		{"@scope/pkg@1.0.0-beta.1", "@scope/pkg", "1.0.0-beta.1", true},
		{"noscope", "", "", false},
		{"@scope/only", "", "", false}, // no version component
	}

	for _, tc := range tests {
		name, version, ok := splitBunPkgKey(tc.key)
		if ok != tc.wantOK {
			t.Errorf("splitBunPkgKey(%q) ok=%v, want %v", tc.key, ok, tc.wantOK)
			continue
		}
		if ok && (name != tc.wantName || version != tc.wantVersion) {
			t.Errorf("splitBunPkgKey(%q) = (%q, %q), want (%q, %q)",
				tc.key, name, version, tc.wantName, tc.wantVersion)
		}
	}
}

// TestCheck_NoManifest confirms that a directory with no manifest returns nil.
func TestCheck_NoManifest(t *testing.T) {
	results, err := Check(t.TempDir(), Options{AgeDays: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results, got %v", results)
	}
}

// TestCheck_RegistryNotFound confirms that 404s are logged as warnings, not errors.
func TestCheck_RegistryNotFound(t *testing.T) {
	// Registry returns 404 for everything.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	lock := `{"lockfileVersion":3,"packages":{"node_modules/ghost-pkg":{"version":"1.0.0"}}}`
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte(lock), 0o644)

	// Should not return an error — 404s are warnings and the package is skipped.
	results, err := Check(dir, Options{AgeDays: 7, RegistryURL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results (package skipped), got %d", len(results))
	}
}
