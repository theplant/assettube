package assettube

import (
	"bytes"
	"net/http"
	"net/http/httptest"

	_ "github.com/bom-d-van/sidekick"

	"testing"
)

func TestManager(t *testing.T) {
	for _, path := range []string{
		"/Users/bom_d_van/Code/go/workspace/src/github.com/theplant/assettube/test",
		"test",
	} {
		m, err := NewManager(path)
		if err != nil {
			t.Fatal(err)
		}
		m.URLPrefix = "/assets"

		for _, c := range [][2]string{
			{"css/file.css", "css/file.0bc77612dba2d5253636e9f0b0d3e6cc.css"},
			{"js/file.js", "js/file.bf5a6a7119046d97ee509d017080c6aa.js"},
		} {
			if got, want := m.AssetsPath(c[0]), c[1]; got != want {
				t.Errorf("m.AssetsPath(%s) = %s; want %s", c[0], got, want)
			}
		}

		req, err := http.NewRequest("GET", "http://example.com/assets/js/file.bf5a6a7119046d97ee509d017080c6aa.js", nil)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		var body bytes.Buffer
		w.Body = &body
		m.ServeHTTP(w, req)
		if got, want := body.String(), "var code = 'test';\n"; got != want {
			t.Errorf("body.String() = %x; want %x", got, want)
		}
	}
}
