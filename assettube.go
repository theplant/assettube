// package assettube fingerprints and servers your assets processed by
// webpack/gulp/other-tools from your Go application.
//
// AssetTube copys your asset files into a subdirectory named `assettube` and
// fingerprints them, in runtime. Every time the server is restarted, it will
// remove previously generated files and generates new files.
package assettube

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash"
	"html"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var DefaultManager, _ = NewManager(Config{})

// Add includes path in Manager serving scope. It also copies and fingerprints
// assets into a subdirectory named "assettube". Every time it's called it
// removes the subdirectory and create a new one, then copy all the matched
// files into the new directory.
func Add(root string) error { return DefaultManager.Add(root) }

// ServeHTTP returns the file content based on URL, stripped of URLPrefix.
func ServeHTTP(w http.ResponseWriter, r *http.Request) { DefaultManager.ServeHTTP(w, r) }

// AssetPath returns the fingerprinted filename, with Hostname and URLPrefix if configured.
// It's mostly used as a template function for package html/template or text/template.
func AssetPath(p string) string { return DefaultManager.AssetPath(p) }

// Integrity returns the SRI hash of corresponding file.
// You could specify which digest hash to use by Config.HashType.
// https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
func Integrity(p string) string { return DefaultManager.Integrity(p) }

// SetConfig updates Manager configurations.
// Under the hood, it creates a new manager and reprocesses all the monitored directories.
func SetConfig(cfg Config) error { return DefaultManager.SetConfig(cfg) }

func Script(p string, attrs ...string) template.HTML { return DefaultManager.Script(p, attrs...) }

func Link(p string, attrs ...string) template.HTML { return DefaultManager.Link(p, attrs...) }

// Manager processes and serves the file content.
type Manager struct {
	paths       []string
	pathsMap    map[string]string
	fpPathsMap  map[string]string
	fingerprint bool

	urlPrefix string
	hostname  string

	hash           HashType
	integrity      bool
	integritiesMap map[string]string

	matcher func(path string, info os.FileInfo) bool

	logger io.Writer
}

type Config struct {
	Fingerprint bool   // Fingerprint option. False by default, so you could only enable it for production server.
	URLPrefix   string // URLPrefix prepends url in the fingerprinted filename returned from Manager.
	Hostname    string // Hostname specifies CDN hostname. Empty by default.

	// Matcher decides what files AssetTube Manager should fingerprint and serve.
	// The default matcher only handle JS and CSS files (i.e.: *.js, *.css).
	Matcher func(path string, info os.FileInfo) bool

	// Enable SubresourceIntegrity support and specify digest hash method by HashType.
	// Note: SubresourceIntegrity only works when Fingerprint is enabled.
	SubresourceIntegrity bool
	HashType             HashType // Default HashType is HTSHA384(SHA-384).

	Logger io.Writer
}

// HashType represents the hash function used in Subresource Integrity.
type HashType int

const (
	// Three hash digest types supported in SubresourceIntegrity.
	HTSHA384 HashType = iota
	HTSHA256
	HTSHA512
)

// Hash returns corresponding Hash functions for checksum calculation.
func (h HashType) Hash() hash.Hash {
	switch h {
	case HTSHA256:
		return sha256.New()
	case HTSHA384:
		return sha512.New384()
	case HTSHA512:
		return sha512.New()
	}
	return md5.New()
}

// Strings return HashType's string name.
func (h HashType) String() string {
	switch h {
	case HTSHA256:
		return "sha256"
	case HTSHA384:
		return "sha384"
	case HTSHA512:
		return "sha512"
	}
	return "md5"
}

// NewManager returns an AssetTube Manager.
func NewManager(cfg Config, paths ...string) (*Manager, error) {
	var m Manager
	m.pathsMap = map[string]string{}
	m.fpPathsMap = map[string]string{}
	m.integritiesMap = map[string]string{}
	m.fingerprint = cfg.Fingerprint
	m.urlPrefix = strings.Trim(cfg.URLPrefix, "/")
	m.hostname = cfg.Hostname
	m.integrity = cfg.SubresourceIntegrity
	m.hash = cfg.HashType

	if cfg.Matcher != nil {
		m.matcher = cfg.Matcher
	} else {
		m.matcher = JSCSSOnly
	}
	if cfg.Logger != nil {
		m.logger = cfg.Logger
	} else {
		m.logger = os.Stdout
	}

	for _, p := range paths {
		if err := m.Add(p); err != nil {
			return nil, err
		}
	}

	return &m, nil
}

func JSCSSOnly(path string, info os.FileInfo) bool {
	if matched := strings.HasSuffix(path, ".js"); matched {
		return true
	} else if matched := strings.HasSuffix(path, ".css"); matched {
		return true
	}
	return false
}

// SetConfig updates Manager configurations. It overrides all configs with the new ones.
// Under the hood, it creates a new manager and reprocesses all the monitored directories.
func (m *Manager) SetConfig(cfg Config) error {
	nm, err := NewManager(cfg, m.paths...)
	if err != nil {
		return err
	}
	*m = *nm
	return nil
}

// Add includes path in Manager serving scope. It also copies and fingerprints
// assets into a subdirectory named "assettube". Every time it's called it
// removes the subdirectory and create a new one, then copy all the matched
// files into the new directory.
func (m *Manager) Add(root string) error {
	defer func(start time.Time) {
		fmt.Fprintf(m.logger, "[AssetTube] %s Add %s took %s\n", time.Now(), root, time.Now().Sub(start))
	}(time.Now())
	m.paths = append(m.paths, root)

	root = filepath.Clean(root)
	cacheDir := root
	if m.fingerprint {
		cacheDir = filepath.Join(root, "assettube")
		if _, err := os.Stat(cacheDir); err != nil {
			if !os.IsNotExist(err) {
				return err
			}
		} else if err := os.RemoveAll(cacheDir); err != nil {
			return err
		}
		if err := os.Mkdir(cacheDir, 0755); err != nil {
			return err
		}
	}

	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := strings.TrimPrefix(strings.TrimPrefix(path, root), string(filepath.Separator))
		if info.IsDir() {
			if name == "" {
				return nil
			} else if name == "assettube" {
				return filepath.SkipDir
			}

			if !m.fingerprint {
				return nil
			}

			if err := os.Mkdir(filepath.Join(cacheDir, name), info.Mode()); err != nil {
				return err
			}
			return nil
		}

		if !m.matcher(path, info) {
			return nil
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		// skip fingerprinting for dev mode
		if !m.fingerprint {
			m.pathsMap[name] = name
			m.fpPathsMap[name] = filepath.Join(cacheDir, name)
			return nil
		}

		// generate fingerprinted filename
		hash := md5.New()
		if _, err := io.Copy(hash, src); err != nil {
			return err
		}
		if _, err := src.Seek(0, 0); err != nil {
			return err
		}
		ext := filepath.Ext(path)
		fpname := fmt.Sprintf("%s.%x%s", strings.TrimSuffix(name, ext), hash.Sum(nil), ext)
		m.pathsMap[name] = fpname
		m.fpPathsMap[fpname] = filepath.Join(cacheDir, fpname)
		if m.integrity {
			hash := m.hash.Hash()
			if _, err := io.Copy(hash, src); err != nil {
				return err
			}
			if _, err := src.Seek(0, 0); err != nil {
				return err
			}
			m.integritiesMap[name] = base64.RawStdEncoding.EncodeToString(hash.Sum(nil))
		}

		// copy file to assettube/
		dst, err := os.OpenFile(m.fpPathsMap[m.pathsMap[name]], os.O_WRONLY|os.O_TRUNC|os.O_CREATE, info.Mode())
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		dst.Close()

		return nil
	}); err != nil {
		return err
	}

	return nil
}

// ServeHTTP returns the file content based on URL, stripped of URLPrefix.
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func(start time.Time) {
		fmt.Fprintf(m.logger, "[AssetTube] %s Serve %s took %s\n", time.Now(), r.URL, time.Now().Sub(start))
	}(time.Now())

	path := r.URL.Path
	if m.urlPrefix != "" {
		path = strings.TrimPrefix(path, "/"+m.urlPrefix)
	}
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		http.NotFound(w, r)
		return
	}

	filePath, exists := m.fpPathsMap[path]
	if !exists {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, filePath)
}

// AssetPath returns the fingerprinted filename, with Hostname and URLPrefix if configured.
// It's mostly used as a template function for package html/template or text/template.
func (m *Manager) AssetPath(p string) string {
	paths := make([]string, 0, 3)
	if m.hostname != "" {
		paths = append(paths, m.hostname)
	}
	if m.urlPrefix != "" {
		paths = append(paths, m.urlPrefix)
	}
	paths = append(paths, m.pathsMap[p])

	if m.hostname != "" {
		return strings.Join(paths, "/")
	}
	return "/" + strings.Join(paths, "/")
}

// Integrity returns the SRI hash of corresponding file.
// You could specify which digest hash to use by Config.HashType.
// https://developer.mozilla.org/en-US/docs/Web/Security/Subresource_Integrity
func (m *Manager) Integrity(p string) string {
	if m.integritiesMap[p] == "" {
		return ""
	}
	return fmt.Sprintf("%s-%s", m.hash, m.integritiesMap[p])
}

func (m *Manager) Script(p string, attrs ...string) template.HTML {
	if m.integrity {
		attrs = append(attrs, "integrity", m.Integrity(p))
	}
	return template.HTML(fmt.Sprintf(`<script src="%s" type="text/javascript"%s></script>`, m.AssetPath(p), stringifyAttrs(attrs)))
}

func (m *Manager) Link(p string, attrs ...string) template.HTML {
	if m.integrity {
		attrs = append(attrs, "integrity", m.Integrity(p))
	}
	return template.HTML(fmt.Sprintf(`<link href="%s" rel="stylesheet" type="text/css"%s></link>`, m.AssetPath(p), stringifyAttrs(attrs)))
}

func stringifyAttrs(attrs []string) string {
	var pairs []string
	for i := 0; i < len(attrs); i += 2 {
		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, html.EscapeString(attrs[i]), html.EscapeString(attrs[i+1])))
	}
	if len(pairs) > 0 {
		pairs = append([]string{""}, pairs...)
	}
	return strings.Join(pairs, " ")
}

func NewManagerManifest(path string) (*Manager, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var m Manager
	var data struct {
		URLPrefix string
		Hostname  string
		Paths     map[string]string
	}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	m.hostname = data.Hostname
	m.urlPrefix = data.URLPrefix
	m.pathsMap = data.Paths
	m.fpPathsMap = map[string]string{}
	m.logger = os.Stdout
	for _, fp := range m.pathsMap {
		m.fpPathsMap[fp] = fp
	}
	return &m, nil
}
