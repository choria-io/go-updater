// Copyright (c) 2018-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package updater

import (
	"errors"
	logger "log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

// Config configures the updater
type Config struct {
	TargetFile      string
	SourceRepo      string
	Version         string
	CurrentVersion  string
	PublicKey       []byte
	Log             logrus.StdLogger
	OperatingSystem string
	Architecture    string
	Downloader      Downloader
}

func newConfig(opts ...Option) (*Config, error) {
	c := &Config{
		TargetFile:      os.Args[0],
		Architecture:    runtime.GOARCH,
		OperatingSystem: runtime.GOOS,
	}

	for _, opt := range opts {
		err := opt(c)
		if err != nil {
			return nil, err
		}
	}

	if c.Downloader == nil {
		err := DownloadMethod(&HTTPDownloader{})(c)
		if err != nil {
			return nil, err
		}
	}

	if c.Log == nil {
		c.Log = logger.New(os.Stdout, "updater:", 0)
	}

	err := c.Validate()
	if err != nil {
		return nil, err
	}

	return c, nil
}

// Validate confirms that the configuration meets the minimum requirements
func (c *Config) Validate() error {
	if c.TargetFile == "" {
		return errors.New("no target file specified, please use TargetFile()")
	}

	if c.SourceRepo == "" {
		return errors.New("no source repo given, please use SourceRepo()")
	}

	if c.Version == "" {
		return errors.New("no version given, please use Version()")
	}

	if c.Log == nil {
		return errors.New("no logger given, please use Logger()")
	}

	return nil
}

// Option is a function that configures the auto updater
type Option func(*Config) error

// Arch overrides the architecture to update to
func Arch(a string) Option {
	return func(c *Config) error {
		c.Architecture = a
		return nil
	}
}

// OS overrides the OS to update to
func OS(os string) Option {
	return func(c *Config) error {
		c.OperatingSystem = os
		return nil
	}
}

// CurrentVersion sets the current version of the app and allows for an early exit if update is not needed, always updates when not set
func CurrentVersion(v string) Option {
	return func(c *Config) error {
		c.CurrentVersion = v
		return nil
	}
}

// Logger sets the logger to use during updates
func Logger(l logrus.StdLogger) Option {
	return func(c *Config) error {
		c.Log = l
		return nil
	}
}

// PublicKey sets the public key to use
func PublicKey(k []byte) Option {
	return func(c *Config) error {
		c.PublicKey = k
		return nil
	}
}

// TargetFile sets the file to update
func TargetFile(f string) Option {
	return func(c *Config) error {
		var err error

		c.TargetFile, err = filepath.Abs(f)

		return err
	}
}

// Version sets the version to deploy
func Version(v string) Option {
	return func(c *Config) error {
		c.Version = v
		return nil
	}
}

// SourceRepo where the releases are found
func SourceRepo(r string) Option {
	return func(c *Config) error {
		c.SourceRepo = r
		return nil
	}
}

// DownloadMethod configures a downloader to use
func DownloadMethod(m Downloader) Option {
	return func(c *Config) error {
		c.Downloader = m
		return c.Downloader.Configure(c)
	}
}
