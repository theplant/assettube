package assettube

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/bom-d-van/sidekick"
)

func TestNewAndAssetPath(t *testing.T) {
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
				{"css/file.css", "/css/file.0bc77612dba2d5253636e9f0b0d3e6cc.css"},
				{"js/file.js", "/js/file.bf5a6a7119046d97ee509d017080c6aa.js"},
			},
			getURL: "http://example.com/js/file.bf5a6a7119046d97ee509d017080c6aa.js",
		},
		{
			cname:       2,
			path:        "test",
			fingerprint: true,
			pathMap: [][2]string{
				{"css/file.css", "/css/file.0bc77612dba2d5253636e9f0b0d3e6cc.css"},
				{"js/file.js", "/js/file.bf5a6a7119046d97ee509d017080c6aa.js"},
			},
			getURL: "http://example.com/js/file.bf5a6a7119046d97ee509d017080c6aa.js",
		},
		{
			cname:       3,
			path:        "test",
			fingerprint: false,
			pathMap: [][2]string{
				{"css/file.css", "/css/file.css"},
				{"js/file.js", "/js/file.js"},
			},
			getURL: "http://example.com/js/file.js",
		},
	} {
		t.Logf("case %d", c.cname)

		m, err := NewManager(Config{Fingerprint: c.fingerprint}, c.path)
		if err != nil {
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
	m, _ := NewManager(Config{Hostname: "https://cdn.com"}, "test")
	if got, want := m.AssetPath("js/file.js"), "https://cdn.com/js/file.js"; got != want {
		t.Errorf("m.AssetPath(js/file.js) = %s; want %s", got, want)
	}
}

func TestIntegrity(t *testing.T) {
	m, err := NewManager(Config{SubresourceIntegrity: true, Fingerprint: true}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := m.Integrity("js/file.js"), "sha384-ikdSg6BDd7ZQH0wpe7EtsWSf4DDnkWmgulB70NrXja4doy1lTsql2ajoHay1xkiu"; got != want {
		t.Errorf(`m.Integrity("js/file.js") = %s; want %s`, got, want)
	}
	if err := m.SetConfig(Config{SubresourceIntegrity: true, Fingerprint: true, HashType: HTSHA512}); err != nil {
		t.Fatal(err)
	}
	if got, want := m.Integrity("js/file.js"), "sha512-ju5jHaaN+e9x7kaWXjRO8fgoYCzKsw7lAzY3uzpjSAF3FJsKoIYAhvZ6Plxp5hgFyu0ho7a7U6mAWxcvKrC+Dw"; got != want {
		t.Errorf(`m.Integrity("js/file.js") = %s; want %s`, got, want)
	}
}

func TestScriptAndLink(t *testing.T) {
	m, err := NewManager(Config{Fingerprint: true}, "test")
	if err != nil {
		t.Error(err)
	}
	if got, want := m.Script("js/file.js"), template.HTML(`<script src="/js/file.bf5a6a7119046d97ee509d017080c6aa.js" type="text/javascript"></script>`); got != want {
		t.Errorf("m.Script(js/file.js) = %s; want %s", got, want)
	}
	if got, want := m.Script("js/file.js", "attr", "val<tag>"), template.HTML(`<script src="/js/file.bf5a6a7119046d97ee509d017080c6aa.js" type="text/javascript" attr="val&lt;tag&gt;"></script>`); got != want {
		t.Errorf("m.Script(js/file.js) = %s; want %s", got, want)
	}
	if got, want := m.Link("css/file.css"), template.HTML(`<link href="/css/file.0bc77612dba2d5253636e9f0b0d3e6cc.css" rel="stylesheet" type="text/css"></link>`); got != want {
		t.Errorf("m.Link(css/file.css) = %s; want %s", got, want)
	}

	if err := m.SetConfig(Config{Fingerprint: true, SubresourceIntegrity: true}); err != nil {
		t.Error(err)
	}
	if got, want := m.Script("js/file.js", "attr", "val<tag>"), template.HTML(`<script src="/js/file.bf5a6a7119046d97ee509d017080c6aa.js" type="text/javascript" attr="val&lt;tag&gt;" integrity="sha384-ikdSg6BDd7ZQH0wpe7EtsWSf4DDnkWmgulB70NrXja4doy1lTsql2ajoHay1xkiu"></script>`); got != want {
		t.Errorf("m.Script(js/file.js) = %s; want %s", got, want)
	}

	if err := m.SetConfig(Config{SubresourceIntegrity: true}); err != nil {
		t.Error(err)
	}
	if got, want := m.Script("js/file.js", "attr", "val<tag>"), template.HTML(`<script src="/js/file.js" type="text/javascript" attr="val&lt;tag&gt;" integrity=""></script>`); got != want {
		t.Errorf("m.Script(js/file.js) = %s; want %s", got, want)
	}
}
