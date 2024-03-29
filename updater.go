// Copyright (c) 2018-2022, R.I. Pienaar and the Choria Project contributors
//
// SPDX-License-Identifier: Apache-2.0

package updater

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"sync"
)

// Spec describes an available package update
type Spec struct {
	BinaryPath string   `json:"binary"`
	BinaryURI  *url.URL `json:"uri,omitempty"`
	Sha256Hash string   `json:"hash"`
	Signature  string   `json:"signature,omitempty"`
}

// Downloader can download releases from repositories
type Downloader interface {
	Configure(*Config) error
	FetchSpec() (*Spec, error)
	FetchBinary(spec *Spec, target string) error
}

var mu = &sync.Mutex{}

// FetchSpec retrieves the available update specification matching opts
func FetchSpec(opts ...Option) (*Spec, error) {
	mu.Lock()
	defer mu.Unlock()

	config, err := newConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid updater configuration: %s", err)
	}

	spec, err := config.Downloader.FetchSpec()
	if err != nil {
		return nil, fmt.Errorf("release %s not found: %s", config.Version, err)
	}

	return spec, nil
}

// Apply applies a specific update
func Apply(opts ...Option) error {
	mu.Lock()
	defer mu.Unlock()

	config, err := newConfig(opts...)
	if err != nil {
		return fmt.Errorf("invalid updater configuration: %s", err)
	}

	spec, err := config.Downloader.FetchSpec()
	if err != nil {
		return fmt.Errorf("release %s not found: %s", config.Version, err)
	}

	config.Log.Printf("Starting update process to %s from %s", config.Version, config.SourceRepo)

	newpath := config.TargetFile + ".new"
	err = config.Downloader.FetchBinary(spec, newpath)
	if err != nil {
		return fmt.Errorf("download failed: %s", err)
	}

	config.Log.Printf("Saved downloaded binary to %s", newpath)

	if !validateChecksum(newpath, spec) {
		return fmt.Errorf("downloaded file had an invalid checksum")
	}

	backup, err := backupTarget(config)
	if err != nil {
		return fmt.Errorf("could not create backup: %s", err)
	}

	config.Log.Printf("Created backup of current binary to %s", backup)

	err = swapNew(newpath, backup, config)

	return err
}

func swapNew(newpath string, backup string, c *Config) error {
	oldpath := fmt.Sprintf("%s.old", c.TargetFile)
	err := os.Rename(c.TargetFile, oldpath)
	if err != nil {
		return errors.New(err.Error())
	}
	defer os.Remove(oldpath)

	err = os.Rename(newpath, c.TargetFile)
	if err != nil {
		rerr := os.Rename(backup, c.TargetFile)
		if rerr != nil {
			return &rollbackErr{err, rerr}
		}

		return err
	}

	return nil
}

func backupTarget(c *Config) (string, error) {
	backuppath := fmt.Sprintf("%s.backup", c.TargetFile)
	stat, err := os.Stat(c.TargetFile)
	if err != nil {
		return "", errors.New(err.Error())
	}

	_ = os.Remove(backuppath)

	out, err := os.OpenFile(backuppath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, stat.Mode())
	if err != nil {
		return "", err
	}
	defer out.Close()

	in, err := os.Open(c.TargetFile)
	if err != nil {
		return "", err
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	return backuppath, err
}

func validateChecksum(newpath string, spec *Spec) bool {
	f, err := os.Open(newpath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}

	sum := fmt.Sprintf("%x", h.Sum(nil))

	return sum == spec.Sha256Hash
}
