package updater

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Updater")
}

var _ = Describe("Apply", func() {
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
	})

	Describe("fetchBinary", func() {

	})
})
