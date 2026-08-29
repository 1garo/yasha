package config

import "os"

type Config struct {
	CassandraHost    string
	CassandraPort    string
	CassandraKeyspace string
}

func Load() Config {
	return Config{
		CassandraHost:    getEnv("CASSANDRA_HOST", "cassandra"),
		CassandraPort:    getEnv("CASSANDRA_PORT", "9042"),
		CassandraKeyspace: getEnv("CASSANDRA_KEYSPACE", "yasha"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}


