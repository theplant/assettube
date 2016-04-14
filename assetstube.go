package assetstube

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
func AssetsPath(p string) string                       { return DefaultManager.AssetsPath(p) }

type Manager struct {
	pathsMap    map[string]string
	fpPathsMap  map[string]string
	URLPrefix   string
	Fingerprint bool
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

// Add includes path in Manager serving scope. It also copys and fingerprints assets into a subdirectory named "assetstube". Everytime it's called it reset the subdirectory and restar
func (m *Manager) Add(root string) error {
	root = filepath.Clean(root)
	cacheDir := filepath.Join(root, "assetstube")
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

	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		name := strings.TrimPrefix(strings.TrimPrefix(path, root), string(filepath.Separator))
		if info.IsDir() {
			if name == "" {
				return nil
			} else if name == "assetstube" {
				return filepath.SkipDir
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
	// if m.Fingerprint {
	// 	return
	// }
	http.ServeFile(w, r, m.fpPathsMap[path])
}

func (m *Manager) AssetsPath(p string) string {
	if m.Hostname != "" {
		return fmt.Sprintf("%s/%s", m.Hostname, m.pathsMap[p])
	}
	return m.pathsMap[p]
}
