package entrecord

import "time"

type Options struct {
	DatabasePath       string
	CacheConnections   bool
	MaxConnections     int
	MaxIdleConnections int
	MaxLifetime        time.Duration
}

func InMemeryOpts(connections int) Options {
	return Options{
		DatabasePath:       ":memory:",
		CacheConnections:   true,
		MaxConnections:     connections,
		MaxIdleConnections: connections,
		MaxLifetime:        0,
	}
}
