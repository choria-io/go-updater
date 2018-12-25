package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	updater "github.com/choria-io/go-updater"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	version string
	osName  string
	arch    string
	root    string
	binary  string
	force   bool
)

func main() {
	app := kingpin.New("update-repo", "The Choria Update repository manager")
	app.Arg("binary", "The binary to add to the repository").Required().ExistingFileVar(&binary)
	app.Flag("version", "The version this binary represents").Required().StringVar(&version)

	app.Flag("arch", "The architecture to add the binary to").Required().StringVar(&arch)
	app.Flag("os", "The operating system to add the binary to").Required().StringVar(&osName)
	app.Flag("repo", "The path to the repository").Default(".").StringVar(&root)
	app.Flag("force", "Overwrite existing files").BoolVar(&force)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	validateBinary()
	validateRepo()

	fname := fmt.Sprintf("%s-%s-%s-%s", filepath.Base(binary), osName, arch, version)
	spec := updater.Spec{
		Sha256Hash: fileSum(binary),
	}

	targetdir := filepath.Join(root, version, osName, arch)
	targetfile := filepath.Join(targetdir, fname)
	spectarget := filepath.Join(targetdir, "release.json")
	compressedfile := targetfile + ".bz2"

	if !force {
		_, err := os.Stat(compressedfile)
		if err == nil {
			kingpin.Fatalf("target %s already exist", compressedfile)
		}
	}

	err := os.MkdirAll(targetdir, 0755)
	kingpin.FatalIfError(err, "could not create target '%s': %s", targetdir, err)

	err = copyfile(binary, targetfile)
	kingpin.FatalIfError(err, "could not copy binary '%s': %s", binary, err)

	if spec.Sha256Hash != fileSum(targetfile) {
		os.Remove(targetfile)
		kingpin.Fatalf("file copy operation did not produce the same checksum")
	}

	err = compress(targetfile)
	kingpin.FatalIfError(err, "could not compress '%s': %s", targetfile, err)

	_, err = os.Stat(compressedfile)
	kingpin.FatalIfError(err, "compression did not create %s", err)

	spec.BinaryPath = fname + ".bz2"

	j, err := json.Marshal(spec)
	kingpin.FatalIfError(err, "could not json encode update spec: %s", err)

	err = ioutil.WriteFile(spectarget, j, 0644)
	kingpin.FatalIfError(err, "could not write spec '%s': %s", spectarget, err)

	fmt.Printf("Copied %s to %s\n", binary, compressedfile)
}

func fileSum(path string) (sum string) {
	f, err := os.Open(path)
	kingpin.FatalIfError(err, "could not open binary '%s': %s", path, err)
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	kingpin.FatalIfError(err, "could not read binary '%s': %s", path, err)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func compress(path string) error {
	var cmd *exec.Cmd

	if force {
		cmd = exec.Command("bzip2", "-f", path)
	} else {
		cmd = exec.Command("bzip2", path)
	}

	return cmd.Run()
}

func validateBinary() {
	_, err := os.Stat(binary)
	kingpin.FatalIfError(err, "the binary '%s' does not exist", binary)
}

func validateRepo() {
	stat, err := os.Stat(root)
	kingpin.FatalIfError(err, "the repository '%s' does not exist", root)
	if !stat.IsDir() {
		kingpin.Fatalf("the repository '%s' is not a directory", root)
	}
}

func copyfile(src string, dst string) error {
	buf := make([]byte, 1024*8)

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		_, err = destination.Write(buf[:n])
		if err != nil {
			return err
		}
	}

	return nil
}
