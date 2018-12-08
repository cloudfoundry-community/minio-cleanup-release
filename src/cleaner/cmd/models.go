package cmd

import "io"

// CleanerConfig is the config
type CleanerConfig struct {
	BaseDirectory string   `toml:"base-directory"`
	Schedule      string   `toml:"schedule"`
	Buckets       []Bucket `toml:"bucket"`
	writer        io.Writer
}

func (c *CleanerConfig) SetWriter(w io.Writer) {
	c.writer = w
}

func (c *CleanerConfig) Writer() io.Writer {
	return c.writer
}

// Bucket is a named list of files
type Bucket struct {
	Name  string `toml:"name"`
	Files []File `toml:"file"`
}

// File has a regexp matching a file name and the number to retain
type File struct {
	Pattern  string `toml:"pattern"`
	Retainer uint   `toml:"retainer"`
}
