package config

type Config struct {
	WorkerCount int    // Number of worker goroutines
	RateLimit   int    // Number of requests per second
	QueueSize   int    // Maximum number of URLs in the queue
	UserAgent   string // User agent string
	MaxDepth    int    // Maximum depth for crawling
}

func DefaultConfig() *Config {
	return &Config{
		WorkerCount: 10,
		RateLimit:   5,
		QueueSize:   100000,
		UserAgent:   "AmberRake",
		MaxDepth:    5,
	}
}

func LowResourceConfig() *Config {
	return &Config{
		WorkerCount: 2,
		RateLimit:   1,
		QueueSize:   1000,
		UserAgent:   "AmberRake",
		MaxDepth:    2,
	}
}

func ProductionConfig() *Config {
	return &Config{
		WorkerCount: 20,
		RateLimit:   10,
		QueueSize:   1000000,
		UserAgent:   "AmberRake",
		MaxDepth:    10,
	}
}

