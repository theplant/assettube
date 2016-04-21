package assettube

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var DefaultManager, _ = NewManager()

func Add(root string) error                            { return DefaultManager.Add(root) }
func ServeHTTP(w http.ResponseWriter, r *http.Request) { DefaultManager.ServeHTTP(w, r) }
func AssetPath(p string) string                        { return DefaultManager.AssetPath(p) }
func SetFingerprint(use bool) error                    { return DefaultManager.SetFingerprint(use) }
func SetURLPrefix(prefix string)                       { DefaultManager.SetURLPrefix(prefix) }
func SetHostname(name string)                          { DefaultManager.SetHostname(name) }

type Manager struct {
	paths       []string
	pathsMap    map[string]string
	fpPathsMap  map[string]string
	fingerprint bool

	urlPrefix string
	hostname  string

	Matcher func(path string, info os.FileInfo) bool
	// TODO: Only []string
	// TODO: Skip []string
}

func NewManager(paths ...string) (*Manager, error) {
	var m Manager
	m.pathsMap = map[string]string{}
	m.fpPathsMap = map[string]string{}
	m.Matcher = defaultMatcher
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

func (m *Manager) SetFingerprint(use bool) error {
	nm, _ := NewManager()
	nm.fingerprint = use
	nm.urlPrefix = m.urlPrefix
	nm.hostname = m.hostname
	nm.Matcher = m.Matcher
	for _, p := range m.paths {
		if err := nm.Add(p); err != nil {
			return err
		}
	}

	*m = *nm
	return nil
}

func (m *Manager) SetURLPrefix(prefix string) { m.urlPrefix = strings.Trim(prefix, "/") }
func (m *Manager) SetHostname(name string)    { m.SetHostname(name) }

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
		ext := filepath.Ext(path)
		fpname := fmt.Sprintf("%s.%x%s", strings.TrimSuffix(name, ext), hash.Sum(nil), ext)
		m.pathsMap[name] = fpname
		m.fpPathsMap[fpname] = filepath.Join(cacheDir, fpname)
		if _, err := src.Seek(0, 0); err != nil {
			return err
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

func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if m.urlPrefix != "" {
		path = strings.TrimPrefix(path, "/"+m.urlPrefix)
	}
	path = strings.TrimPrefix(path, "/")
	http.ServeFile(w, r, m.fpPathsMap[path])
}

func (m *Manager) AssetPath(p string) string {
	paths := make([]string, 0, 3)
	if m.hostname != "" {
		paths = append(paths, m.hostname)
	}
	if m.urlPrefix != "" {
		paths = append(paths, m.urlPrefix)
	}
	// if m.fingerprint {
	// 	paths = append(paths, "assettube")
	// }
	paths = append(paths, m.pathsMap[p])

	if m.hostname != "" {
		return strings.Join(paths, "/")
	}
	return "/" + strings.Join(paths, "/")
}
