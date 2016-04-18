package assettube

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/bom-d-van/sidekick"
)

func TestManager(t *testing.T) {
	for _, c := range []struct {
		cname       int
		path        string
		fingerprint bool
		pathMap     [][2]string
		getURL      string
	}{
		{
			cname:       1,
			path:        "/Users/bom_d_van/Code/go/workspace/src/github.com/theplant/assettube/test",
			fingerprint: true,
			pathMap: [][2]string{
				{"css/file.css", "assettube/css/file.0bc77612dba2d5253636e9f0b0d3e6cc.css"},
				{"js/file.js", "assettube/js/file.bf5a6a7119046d97ee509d017080c6aa.js"},
			},
			getURL: "http://example.com/assettube/js/file.bf5a6a7119046d97ee509d017080c6aa.js",
		},
		{
			cname:       2,
			path:        "test",
			fingerprint: true,
			pathMap: [][2]string{
				{"css/file.css", "assettube/css/file.0bc77612dba2d5253636e9f0b0d3e6cc.css"},
				{"js/file.js", "assettube/js/file.bf5a6a7119046d97ee509d017080c6aa.js"},
			},
			getURL: "http://example.com/assettube/js/file.bf5a6a7119046d97ee509d017080c6aa.js",
		},
		{
			cname:       3,
			path:        "test",
			fingerprint: false,
			pathMap: [][2]string{
				{"css/file.css", "css/file.css"},
				{"js/file.js", "js/file.js"},
			},
			getURL: "http://example.com/assettube/js/file.js",
		},
	} {
		t.Logf("case %d", c.cname)

		m, err := NewManager(c.path)
		if err != nil {
			t.Fatal(err)
		}
		// m.URLPrefix = "/assets"
		if err := m.UseFingerprint(c.fingerprint); err != nil {
			t.Fatal(err)
		}

		for _, c := range c.pathMap {
			if got, want := m.AssetPath(c[0]), c[1]; got != want {
				t.Errorf("m.AssetPath(%s) = %s; want %s", c[0], got, want)
			}
		}

		req, err := http.NewRequest("GET", c.getURL, nil)
		if err != nil {
			t.Fatal(err)
		}
		w := httptest.NewRecorder()
		var body bytes.Buffer
		w.Body = &body
		m.ServeHTTP(w, req)
		if got, want := body.String(), "var code = 'test';\n"; got != want {
			t.Errorf("body.String() = %s; want %s", got, want)
		}
	}
}

func TestHostname(t *testing.T) {
	m, _ := NewManager("test")
	m.Hostname = "https://cdn.com"
	if got, want := m.AssetPath("js/file.js"), "https://cdn.com/js/file.js"; got != want {
		t.Errorf("m.AssetPath(js/file.js) = %s; want %s", got, want)
	}
}
