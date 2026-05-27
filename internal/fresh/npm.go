package fresh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	registryURL    = "https://registry.npmjs.org"
	requestTimeout = 10 * time.Second
	maxConcurrent  = 10
)

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
	return fetchAll(deps, opts)
}

// dep is a resolved name → exact version pair.
type dep struct {
	name    string
	version string // empty means "fetch latest from registry"
}

func resolveDeps(path string, includeDev bool) ([]dep, error) {
	if _, err := os.Stat(filepath.Join(path, "package-lock.json")); err == nil {
		return parseLock(filepath.Join(path, "package-lock.json"), includeDev)
	}
	if _, err := os.Stat(filepath.Join(path, "package.json")); err == nil {
		return parsePkg(filepath.Join(path, "package.json"), includeDev)
	}
	return nil, nil
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
			name:    strings.TrimPrefix(key, "node_modules/"),
			version: pkg.Version,
		})
	}
	return deps, nil
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
			fmt.Fprintf(os.Stderr, "warn: %s: %v\n", w.pkg, w.err)
			continue
		}
		results = append(results, w.res)
	}
	return results, nil
}

func fetchOne(d dep, opts Options) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	url := opts.baseURL() + "/" + d.name
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json")

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
		// Abbreviated metadata omits time; retry with full document.
		return fetchOneFull(d.name, version, opts)
	}

	return buildResult(d.name, version, rawTime, opts.AgeDays)
}

// fetchOneFull fetches the full registry document to get the time field.
func fetchOneFull(name, version string, opts Options) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, opts.baseURL()+"/"+name, nil)
	if err != nil {
		return Result{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Result{}, fmt.Errorf("registry request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{}, fmt.Errorf("registry returned %d", resp.StatusCode)
	}

	var meta npmMeta
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return Result{}, fmt.Errorf("decode registry response: %w", err)
	}

	rawTime, ok := meta.Time[version]
	if !ok {
		return Result{}, fmt.Errorf("version %s not found in registry time map", version)
	}

	return buildResult(name, version, rawTime, opts.AgeDays)
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
