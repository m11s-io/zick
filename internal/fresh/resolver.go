package fresh

import "time"

type Risk int

const (
	RiskOK   Risk = iota
	RiskWarn      // within age gate
	RiskHigh      // within half the age gate
)

type Result struct {
	Package   string
	Version   string
	Published time.Time
	Age       time.Duration
	Risk      Risk
}

type Options struct {
	AgeDays     int
	IncludeDev  bool
	RegistryURL string // empty → default npm registry; set in tests to inject a mock server
}

func (o Options) baseURL() string {
	if o.RegistryURL != "" {
		return o.RegistryURL
	}
	return registryURL
}

func classify(published time.Time, ageDays int) Risk {
	days := time.Since(published).Hours() / 24
	switch {
	case days < float64(ageDays)/2:
		return RiskHigh
	case days < float64(ageDays):
		return RiskWarn
	default:
		return RiskOK
	}
}
