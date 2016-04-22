package assettube

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var DefaultManager, _ = NewManager(Config{})

func Add(root string) error                            { return DefaultManager.Add(root) }
func ServeHTTP(w http.ResponseWriter, r *http.Request) { DefaultManager.ServeHTTP(w, r) }
func AssetPath(p string) string                        { return DefaultManager.AssetPath(p) }
func Integrity(p string) string                        { return DefaultManager.Integrity(p) }
func SetConfig(cfg Config) error                       { return DefaultManager.SetConfig(cfg) }

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

	Matcher func(path string, info os.FileInfo) bool
}

type Config struct {
	Fingerprint bool
	URLPrefix   string
	Hostname    string
	Matcher     func(path string, info os.FileInfo) bool

	// Enable SubresourceIntegrity support and specify digest hash method by HashType
	SubresourceIntegrity bool
	HashType             HashType
}

type HashType int

const (
	HTMD5 HashType = iota
	HTSHA256
	HTSHA384
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

func NewManager(cfg Config, paths ...string) (*Manager, error) {
	var m Manager
	m.pathsMap = map[string]string{}
	m.fpPathsMap = map[string]string{}
	m.integritiesMap = map[string]string{}
	m.fingerprint = cfg.Fingerprint
	m.urlPrefix = cfg.URLPrefix
	m.hostname = cfg.Hostname
	m.integrity = cfg.SubresourceIntegrity
	m.hash = cfg.HashType
	if m.integrity && m.hash == HTMD5 {
		m.hash = HTSHA384
	}
	if cfg.Matcher != nil {
		m.Matcher = cfg.Matcher
	} else {
		m.Matcher = defaultMatcher
	}

	for _, p := range paths {
		if err := m.Add(p); err != nil {
			return nil, err
		}
	}

	return &m, nil
}

func defaultMatcher(path string, info os.FileInfo) bool {
	if matched := strings.HasSuffix(path, ".js"); matched {
		return true
	} else if matched := strings.HasSuffix(path, ".css"); matched {
		return true
	}
	return false
}

// SetConfig updates Manager config.
// Under the hood, it creates a new manager and reprocesses all the monitored directories.
func (m *Manager) SetConfig(cfg Config) error {
	nm, err := NewManager(cfg, m.paths...)
	if err != nil {
		return err
	}
	*m = *nm
	return nil
}

// Add includes path in Manager serving scope. It also copys and fingerprints
// assets into a subdirectory named "assettube". Everytime it's called it
// reset the subdirectory and restar
func (m *Manager) Add(root string) error {
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

		if !m.Matcher(path, info) {
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

// ServeHTTP returns the fiel content based on url, stripped of URLPrefix.
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if m.urlPrefix != "" {
		path = strings.TrimPrefix(path, "/"+m.urlPrefix)
	}
	path = strings.TrimPrefix(path, "/")
	http.ServeFile(w, r, m.fpPathsMap[path])
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
func (m *Manager) Integrity(p string) string {
	if m.integritiesMap[p] == "" {
		return ""
	}
	return fmt.Sprintf("%s-%s", m.hash, m.integritiesMap[p])
}
