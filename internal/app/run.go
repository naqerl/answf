package app

import (
	"time"

	"github.com/naqerl/answf/internal/cache"
	"github.com/naqerl/answf/internal/cli"
	"github.com/naqerl/answf/internal/fetch"
	"github.com/naqerl/answf/internal/search"
)

func Run(cfg cli.Config) (string, error) {
	c := cache.Manager{
		Dir:      cfg.CacheDir,
		Disabled: cfg.NoCache,
		Now:      time.Now,
	}

	if cfg.Search != "" {
		return search.Run(search.Config{
			Query:    cfg.Search,
			SearXURL: cfg.SearXURL,
			Timeout:  time.Duration(cfg.TimeoutMS) * time.Millisecond,
			Verbose:  cfg.Verbose,
			Top:      cfg.Top,
		}, c)
	}

	return fetch.Run(fetch.Config{
		TargetURL:       cfg.TargetURL,
		WSEndpoint:      cfg.WSEndpoint,
		Timeout:         time.Duration(cfg.TimeoutMS) * time.Millisecond,
		Markdown:        cfg.Markdown,
		FallbackTextise: cfg.FallbackTextise,
		TextiseBaseURL:  cfg.TextiseBaseURL,
	}, c)
}
