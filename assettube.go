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
func UseFingerprint(use bool) error                    { return DefaultManager.UseFingerprint(use) }

type Manager struct {
	paths       []string
	pathsMap    map[string]string
	fpPathsMap  map[string]string
	URLPrefix   string
	fingerprint bool
	Hostname    string

	// TODO: Matcher func
	// TODO: Only []string
	// TODO: Skip []string
}

func NewManager(paths ...string) (*Manager, error) {
	var m Manager
	m.pathsMap = map[string]string{}
	m.fpPathsMap = map[string]string{}
	for _, p := range paths {
		if err := m.Add(p); err != nil {
			return nil, err
		}
	}

	return &m, nil
}

func (m *Manager) UseFingerprint(use bool) error {
	nm, _ := NewManager()
	nm.fingerprint = use
	for _, p := range m.paths {
		if err := nm.Add(p); err != nil {
			return err
		}
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

		if !m.fingerprint {
			m.pathsMap[name] = name
			m.fpPathsMap[name] = filepath.Join(cacheDir, name)
			return nil
		}

		if ext := filepath.Ext(path); ext != "" {
			hash := md5.New()
			if _, err := io.Copy(hash, src); err != nil {
				return err
			}
			fpname := fmt.Sprintf("%s.%x%s", strings.TrimSuffix(name, ext), hash.Sum(nil), ext)
			m.pathsMap[name] = fpname
			m.fpPathsMap[fpname] = filepath.Join(cacheDir, fpname)

			if _, err := src.Seek(0, 0); err != nil {
				return err
			}
		} else {
			// no fingerprint for files without extention. odd behaviour?
			m.pathsMap[name] = name
			m.fpPathsMap[name] = filepath.Join(cacheDir, name)
		}

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
	path := strings.TrimPrefix(r.URL.Path, m.URLPrefix)
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimPrefix(path, "assettube/")
	http.ServeFile(w, r, m.fpPathsMap[path])
}

func (m *Manager) AssetPath(p string) string {
	if m.Hostname != "" {
		return fmt.Sprintf("%s/%s", m.Hostname, m.pathsMap[p])
	}
	if !m.fingerprint {
		return p
	}
	return fmt.Sprintf("assettube/%s", m.pathsMap[p])
}
