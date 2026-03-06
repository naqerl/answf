package cli

type Config struct {
	FetchURL        string
	Search          string
	TargetURL       string
	Markdown        bool
	WSEndpoint      string
	SearXURL        string
	TimeoutMS       float64
	FallbackTextise bool
	TextiseBaseURL  string
	Verbose         bool
	Top             int
	CacheDir        string
	NoCache         bool
}
