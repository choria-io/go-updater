package updater

import (
	"compress/bzip2"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

// HTTPDownloader downloads releases from a normal HTTP(S) server
//
// The update directory structure is:
//
//   root/<Version>/<OperatingSystem>/<Architecture>/release.json
type HTTPDownloader struct {
	cfg *Config
}

// Configure implements Downloader
func (h *HTTPDownloader) Configure(c *Config) error {
	h.cfg = c
	return nil
}

// FetchSpec implements Downloader
func (h *HTTPDownloader) FetchSpec() (spec *Spec, err error) {
	c := h.cfg

	specuri := fmt.Sprintf("%s/%s/%s/%s/release.json", c.SourceRepo, c.Version, c.OperatingSystem, c.Architecture)

	resp, err := http.Get(specuri)
	if err != nil {
		return nil, fmt.Errorf("could not download release spec: %s", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not fetch release from %s: %s", specuri, resp.Status)
	}

	specj, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read repo response: %s", err)
	}

	spec = &Spec{}
	err = json.Unmarshal(specj, spec)
	if err != nil {
		return nil, fmt.Errorf("could not parse spec %s: %s", specuri, err)
	}

	spec.BinaryURI, err = url.Parse(fmt.Sprintf("%s/%s/%s/%s/%s", c.SourceRepo, c.Version, c.OperatingSystem, c.Architecture, spec.BinaryPath))
	if err != nil {
		return nil, fmt.Errorf("could not construct full path to the binary: %s", err)
	}

	return spec, nil
}

// FetchBinary implements Downloader
func (h *HTTPDownloader) FetchBinary(spec *Spec, target string) error {
	stat, err := os.Stat(h.cfg.TargetFile)
	if err != nil {
		return err
	}
	outf, err := os.Create(target)
	if err != nil {
		return err
	}
	defer outf.Close()

	h.cfg.Log.Printf("Fetching %s", spec.BinaryURI)

	resp, err := http.Get(spec.BinaryURI.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("could not download binary: %s", resp.Status)
	}

	n, err := io.Copy(outf, bzip2.NewReader(resp.Body))
	if err != nil {
		return fmt.Errorf("could not save file: %s", err)
	}

	h.cfg.Log.Printf("Fetched %d bytes from %s", n, spec.BinaryURI.String())

	tf := fmt.Sprintf("%s.new", h.cfg.TargetFile)
	err = os.Rename(outf.Name(), tf)
	if err != nil {
		return fmt.Errorf("could not move temporary file to taget: %s", err)
	}

	err = os.Chmod(tf, stat.Mode())
	if err != nil {
		return fmt.Errorf("could not set new file modes: %s", err)
	}

	return nil
}
