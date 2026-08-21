package http

// ExportDefaultMaxIdleConnsPerHost returns the default MaxIdleConnsPerHost value
// from the default client configuration. This function exists solely for testing purposes.
func ExportDefaultMaxIdleConnsPerHost() int {
	cfg := defaultConfig()
	return cfg.maxIdleConnsPerHost
}
