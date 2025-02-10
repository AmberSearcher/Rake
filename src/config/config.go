package config

type Config struct {
	WorkerCount int
	RateLimit   int
	QueueSize   int
	UserAgent   string
}

func DefaultConfig() *Config {
	return &Config{
		WorkerCount: 10,
		RateLimit:   5,
		QueueSize:   100000,
		UserAgent:   "AmberRake",
	}
}

func LowResourceConfig() *Config {
	return &Config{
		WorkerCount: 2,
		RateLimit:   1,
		QueueSize:   1000,
		UserAgent:   "AmberRake",
	}
}

func ProductionConfig() *Config {
	return &Config{
		WorkerCount: 20,
		RateLimit:   10,
		QueueSize:   1000000,
		UserAgent:   "AmberRake",
	}
}
