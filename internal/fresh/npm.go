package fresh

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	registryURL    = "https://registry.npmjs.org"
	requestTimeout = 10 * time.Second
	maxConcurrent  = 10
)

// trailingCommaRe strips trailing commas before ] or } so bun.lock (JSONC) can
// be parsed by the standard library. This is safe for bun.lock because the
// string values it contains (URLs, integrity hashes) never themselves contain ,}
// or ,] sequences.
var trailingCommaRe = regexp.MustCompile(`,(\s*[}\]])`)

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type packageLockV2 struct {
	LockfileVersion int `json:"lockfileVersion"`
	Packages        map[string]struct {
		Version string `json:"version"`
		Dev     bool   `json:"dev"`
	} `json:"packages"`
}

type bunLock struct {
	// Packages maps keys to [name@version, url, {deps}, integrity] arrays.
	// The key may be the package name, "name@version", or a resolution path
	// like "parent-pkg/dep-pkg". The first array element is always the
	// canonical "name@version" string and is the only reliable source.
	// Dev/prod distinction is not available at this level — --include-dev is
	// a no-op when bun.lock is the source.
	Packages map[string][]json.RawMessage `json:"packages"`
}

// npmMeta is the subset of registry metadata we need.
type npmMeta struct {
	DistTags map[string]string `json:"dist-tags"`
	Time     map[string]string `json:"time"`
}

// Check resolves dependencies at path and checks freshness against opts.AgeDays.
func Check(path string, opts Options) ([]Result, error) {
	deps, err := resolveDeps(path, opts.IncludeDev)
	if err != nil {
		return nil, err
	}
	if len(deps) == 0 {
		return nil, nil
	}
	results, err := fetchAll(deps, opts)
	if err != nil {
		return nil, err
	}
	sort.Slice(results, func(i, j int) bool {
		if results[i].Risk != results[j].Risk {
			return results[i].Risk > results[j].Risk
		}
		if results[i].Package != results[j].Package {
			return results[i].Package < results[j].Package
		}
		return results[i].Version < results[j].Version
	})
	return results, nil
}

// dep is a resolved name → exact version pair.
type dep struct {
	name    string
	version string // empty means "fetch latest from registry"
}

func resolveDeps(path string, includeDev bool) ([]dep, error) {
	// bun.lock covers the entire workspace with exact versions — prefer it.
	if _, err := os.Stat(filepath.Join(path, "bun.lock")); err == nil {
		return parseBunLock(filepath.Join(path, "bun.lock"))
	}
	if _, err := os.Stat(filepath.Join(path, "pnpm-lock.yaml")); err == nil {
		return parsePnpmLock(filepath.Join(path, "pnpm-lock.yaml"))
	}
	if _, err := os.Stat(filepath.Join(path, "yarn.lock")); err == nil {
		return parseYarnLock(filepath.Join(path, "yarn.lock"))
	}
	if _, err := os.Stat(filepath.Join(path, "package-lock.json")); err == nil {
		return parseLock(filepath.Join(path, "package-lock.json"), includeDev)
	}
	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		return parsePkg(filepath.Join(path, "package.json"), includeDev)
	}
	return nil, nil
}

// parseBunLock extracts exact package versions from a bun.lock file.
// bun.lock is JSONC (trailing commas allowed) and the packages section key
// format is "name@version" — one entry per resolved package across all
// workspaces. --include-dev has no effect here since bun.lock does not
// distinguish dev from prod at the packages level.
func parseBunLock(path string) ([]dep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	cleaned := trailingCommaRe.ReplaceAll(data, []byte("$1"))

	var lock bunLock
	if err := json.Unmarshal(cleaned, &lock); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	seen := make(map[string]struct{}, len(lock.Packages))
	var deps []dep
	for _, arr := range lock.Packages {
		if len(arr) == 0 {
			continue
		}
		// First element is always the canonical "name@version" string.
		var nameAtVersion string
		if err := json.Unmarshal(arr[0], &nameAtVersion); err != nil {
			continue
		}
		name, version, ok := splitBunPkgKey(nameAtVersion)
		if !ok {
			continue
		}
		// Deduplicate: the same resolved package may appear under multiple
		// keys (direct dep + path-based resolution for the same version).
		if _, dup := seen[nameAtVersion]; dup {
			continue
		}
		seen[nameAtVersion] = struct{}{}
		deps = append(deps, dep{name: name, version: version})
	}
	return deps, nil
}

// splitBunPkgKey splits a bun.lock package key like "lodash@4.17.21" or
// "@types/react@19.2.14" into (name, version). Returns ok=false if the key
// has no version component.
func splitBunPkgKey(key string) (name, version string, ok bool) {
	// Find the last "@" — for scoped packages like "@types/react@1.0.0" the
	// first "@" is the scope prefix, so we must use the last one.
	idx := strings.LastIndex(key, "@")
	if idx <= 0 {
		return "", "", false
	}
	return key[:idx], key[idx+1:], true
}

func parseLock(path string, includeDev bool) ([]dep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var lock packageLockV2
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var deps []dep
	for key, pkg := range lock.Packages {
		if !strings.HasPrefix(key, "node_modules/") {
			continue
		}
		if pkg.Dev && !includeDev {
			continue
		}
		deps = append(deps, dep{
			name:    packageNameFromLockKey(key),
			version: pkg.Version,
		})
	}
	return deps, nil
}

func packageNameFromLockKey(key string) string {
	parts := strings.Split(key, "node_modules/")
	return parts[len(parts)-1]
}

func parsePkg(path string, includeDev bool) ([]dep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	// No lockfile: version is left empty; fetchOne will use dist-tags.latest.
	var deps []dep
	for name := range pkg.Dependencies {
		deps = append(deps, dep{name: name})
	}
	if includeDev {
		for name := range pkg.DevDependencies {
			deps = append(deps, dep{name: name})
		}
	}
	return deps, nil
}

type pnpmLock struct {
	Packages map[string]any `yaml:"packages"`
}

func parsePnpmLock(path string) ([]dep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var lock pnpmLock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	seen := make(map[string]struct{}, len(lock.Packages))
	deps := make([]dep, 0, len(lock.Packages))
	for key := range lock.Packages {
		name, version, ok := splitPnpmPackageKey(key)
		if !ok {
			continue
		}
		id := name + "@" + version
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		deps = append(deps, dep{name: name, version: version})
	}
	return deps, nil
}

func splitPnpmPackageKey(key string) (name, version string, ok bool) {
	key = strings.TrimPrefix(key, "/")
	if key == "" || strings.HasPrefix(key, "file:") || strings.HasPrefix(key, "link:") {
		return "", "", false
	}
	if idx := strings.Index(key, "("); idx >= 0 {
		key = key[:idx]
	}

	idx := strings.LastIndex(key, "@")
	if idx <= 0 || idx == len(key)-1 {
		return "", "", false
	}
	return key[:idx], key[idx+1:], true
}

func parseYarnLock(path string) ([]dep, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var deps []dep
	var selectors []string
	seen := map[string]struct{}{}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(raw, " ") && strings.HasSuffix(line, ":") {
			selectors = parseYarnSelectors(strings.TrimSuffix(line, ":"))
			continue
		}
		if strings.HasPrefix(line, "version ") && len(selectors) > 0 {
			version := strings.Trim(strings.TrimPrefix(line, "version "), `"`)
			for _, selector := range selectors {
				name, ok := packageNameFromYarnSelector(selector)
				if !ok {
					continue
				}
				id := name + "@" + version
				if _, dup := seen[id]; dup {
					continue
				}
				seen[id] = struct{}{}
				deps = append(deps, dep{name: name, version: version})
			}
			selectors = nil
		}
	}
	return deps, nil
}

func parseYarnSelectors(value string) []string {
	parts := strings.Split(value, ",")
	selectors := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.Trim(strings.TrimSpace(part), `"`)
		if part != "" {
			selectors = append(selectors, part)
		}
	}
	return selectors
}

func packageNameFromYarnSelector(selector string) (string, bool) {
	idx := strings.LastIndex(selector, "@")
	if idx <= 0 {
		return "", false
	}
	return selector[:idx], true
}

func fetchAll(deps []dep, opts Options) ([]Result, error) {
	type work struct {
		res Result
		err error
		pkg string
	}

	sem := make(chan struct{}, maxConcurrent)
	ch := make(chan work, len(deps))
	var wg sync.WaitGroup

	for _, d := range deps {
		wg.Add(1)
		go func(d dep) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res, err := fetchOne(d, opts)
			ch <- work{res: res, err: err, pkg: d.name}
		}(d)
	}

	wg.Wait()
	close(ch)

	var results []Result
	for w := range ch {
		if w.err != nil {
			fmt.Fprintf(warnWriter(opts), "warn: %s: %v\n", w.pkg, w.err)
			continue
		}
		results = append(results, w.res)
	}
	return results, nil
}

func warnWriter(opts Options) io.Writer {
	if opts.ErrOut != nil {
		return opts.ErrOut
	}
	return os.Stderr
}

func fetchOne(d dep, opts Options) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	// Fetch the full packument — the abbreviated vnd.npm.install-v1+json format
	// intentionally omits the "time" field, which is the only data we need.
	// One full request is faster than abbreviated + fallback.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.baseURL()+"/"+d.name, nil)
	if err != nil {
		return Result{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("registry request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return Result{}, fmt.Errorf("package not found in registry (private?)")
	}
	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("registry returned %d", resp.StatusCode)
	}

	var meta npmMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return Result{}, fmt.Errorf("decode registry response: %w", err)
	}

	version := d.version
	if version == "" {
		version = meta.DistTags["latest"]
	}

	rawTime, ok := meta.Time[version]
	if !ok {
		return Result{}, fmt.Errorf("version %s not found in registry time map", version)
	}

	return buildResult(d.name, version, rawTime, opts.AgeDays)
}

func buildResult(name, version, rawTime string, ageDays int) (Result, error) {
	published, err := time.Parse(time.RFC3339, rawTime)
	if err != nil {
		// Some entries use milliseconds without a timezone suffix.
		published, err = time.Parse("2006-01-02T15:04:05.000Z", rawTime)
		if err != nil {
			return Result{}, fmt.Errorf("parse publish time %q: %w", rawTime, err)
		}
	}

	age := time.Since(published)
	return Result{
		Package:   name,
		Version:   version,
		Published: published,
		Age:       age,
		Risk:      classify(published, ageDays),
	}, nil
}
