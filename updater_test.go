package updater

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Updater")
}

var _ = Describe("Updater", func() {
	Describe("FetchSpec", func() {
		It("Should fetch the correct spec", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/0.7.0/linux/amd64/release.json"))
				spec, _ := ioutil.ReadFile("testdata/0.7.0/linux/amd64/release.json")
				fmt.Fprint(w, string(spec))
			}))
			defer ts.Close()

			spec, err := FetchSpec(SourceRepo(ts.URL), Version("0.7.0"), OS("linux"), Arch("amd64"))
			Expect(err).ToNot(HaveOccurred())
			Expect(spec.BinaryPath).To(Equal("choria.bz2"))
			Expect(spec.Sha256Hash).To(Equal("12a61f4e173fb3a11c05d6471f74728f76231b4a5fcd9667cef3af87a3ae4dc2"))
			Expect(spec.BinaryURI.String()).To(Equal(fmt.Sprintf("%s/0.7.0/linux/amd64/choria.bz2", ts.URL)))
		})

		It("Should detect bad configs", func() {
			_, err := FetchSpec()
			Expect(err).To(MatchError("invalid updater configuration: no source repo given, please use SourceRepo()"))
		})
	})

	Describe("Apply", func() {
		It("Should download and apply the update", func() {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				f, err := ioutil.ReadFile(filepath.Join("testdata", r.URL.Path))
				Expect(err).ToNot(HaveOccurred())
				fmt.Fprint(w, string(f))
			}))
			defer ts.Close()

			err := ioutil.WriteFile("testdata/target", []byte("target file"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/target")

			err = Apply(
				SourceRepo(ts.URL),
				Version("0.7.0"),
				OS("linux"),
				Arch("amd64"),
				TargetFile("testdata/target"),
				Logger(log.New(ioutil.Discard, "", 0)),
			)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/target.backup")

			swapped, err := ioutil.ReadFile("testdata/target")
			Expect(err).ToNot(HaveOccurred())
			Expect(swapped).To(Equal([]byte("testing\n")))

			backup, err := ioutil.ReadFile("testdata/target.backup")
			Expect(err).ToNot(HaveOccurred())
			Expect(backup).To(Equal([]byte("target file")))
		})
	})

	Describe("swapNew", func() {
		It("Should detect rename errors", func() {
			err := swapNew("/nonexisting.new", "/nonexisting.backup", &Config{TargetFile: "/nonexisting"})
			Expect(err).To(MatchError("rename /nonexisting /nonexisting.old: no such file or directory"))
		})

		It("Should attempt to recover from renaming the new to the target and wrap the recovery error in a rollback error", func() {
			err := ioutil.WriteFile("testdata/source", []byte("old file"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/source")

			err = swapNew("/nonexisting", "/nonexisting", &Config{TargetFile: "testdata/source"})
			err = RollbackError(err)
			Expect(err).To(MatchError("rename /nonexisting testdata/source: no such file or directory"))
		})

		It("Should recover from rename errors by copying the backup data", func() {
			err := ioutil.WriteFile("testdata/source", []byte("old file"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/source")

			err = ioutil.WriteFile("testdata/backup", []byte("backup file"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/backup")

			err = swapNew("testdata/new", "testdata/backup", &Config{TargetFile: "testdata/source"})
			Expect(err).To(HaveOccurred())

			recovered, err := ioutil.ReadFile("testdata/source")
			Expect(err).ToNot(HaveOccurred())
			Expect(recovered).To(Equal([]byte("backup file")))
		})

		It("Should swap the file", func() {
			err := ioutil.WriteFile("testdata/source", []byte("old file"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/source")

			err = ioutil.WriteFile("testdata/backup", []byte("backup file"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/backup")

			err = ioutil.WriteFile("testdata/new", []byte("new file"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/new")

			err = swapNew("testdata/new", "testdata/backup", &Config{TargetFile: "testdata/source"})
			Expect(err).ToNot(HaveOccurred())

			swapped, err := ioutil.ReadFile("testdata/source")
			Expect(err).ToNot(HaveOccurred())
			Expect(swapped).To(Equal([]byte("new file")))
		})
	})

	Describe("backupTarget", func() {
		It("Should detect missing targets", func() {
			_, err := backupTarget(&Config{TargetFile: "/noexisting"})
			Expect(err).To(MatchError("stat /noexisting: no such file or directory"))
		})

		It("Should create the backup", func() {
			err := ioutil.WriteFile("testdata/source", []byte("example data"), 0600)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove("testdata/source")

			out, err := backupTarget(&Config{TargetFile: "testdata/source"})
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(out)
			Expect(out).To(Equal("testdata/source.backup"))

			src, err := ioutil.ReadFile("testdata/source")
			Expect(err).ToNot(HaveOccurred())
			copy, err := ioutil.ReadFile(out)
			Expect(err).ToNot(HaveOccurred())
			Expect(src).To(Equal(copy))
		})
	})
	Describe("validateChecksum", func() {
		It("Should correctly validate checksums", func() {
			s := &Spec{
				Sha256Hash: "f5b72762ab4080e712e266825400a63da7df57c31e485013d8f03070a631aee1",
			}

			Expect(validateChecksum("testdata/0.7.0/linux/amd64/choria.bz2", s)).To(BeTrue())
			s.Sha256Hash = "fail"
			Expect(validateChecksum("testdata/0.7.0/linux/amd64/choria.bz2", s)).To(BeFalse())
		})
	})
})
