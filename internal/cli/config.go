package cli

type Config struct {
	ConfigPath          string
	FetchURL            string
	Search              string
	TargetURL           string
	Markdown            bool
	PlaywrightURL       string
	SearXURL            string
	PlaywrightTimeoutMS float64
	SearchTimeoutMS     float64
	FallbackTextise     bool
	TextiseBaseURL      string
	Verbose             bool
	Top                 int
	CacheDir            string
	NoCache             bool
}
