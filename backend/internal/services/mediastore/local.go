package mediastore

import (
	"io"
	"os"
	"path/filepath"
)

// BlobStorage exposes the S3-compatible interface for all media handles
type BlobStorage interface {
	Save(id string, reader io.Reader) (string, error)
	Delete(id string) error
	GetURL(id string) string
}

type LocalStorage struct {
	baseDir string
	baseURL string
}

func NewLocalStorage(baseDir string, baseURL string) *LocalStorage {
	return &LocalStorage{
		baseDir: baseDir,
		baseURL: baseURL,
	}
}

func (s *LocalStorage) Save(id string, reader io.Reader) (string, error) {
	path := filepath.Join(s.baseDir, id)
	
	outFile, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, reader); err != nil {
		return "", err
	}

	return path, nil
}

func (s *LocalStorage) Delete(id string) error {
	path := filepath.Join(s.baseDir, id)
	return os.Remove(path)
}

// GetURL returns the accessible URL for the media asset.
// Example: baseURL could be "/media" mapping to a static Echo route
func (s *LocalStorage) GetURL(id string) string {
	return s.baseURL + "/" + id
}
