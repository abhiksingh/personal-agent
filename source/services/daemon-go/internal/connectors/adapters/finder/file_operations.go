package finder

import (
	"fmt"
	"os"
	"path/filepath"
)

type pathInfo struct {
	IsDir     bool
	SizeBytes int64
}

func listPath(targetPath string) (int, bool, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("stat target path: %w", err)
	}
	if !info.IsDir() {
		return 1, true, nil
	}
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return 0, true, fmt.Errorf("read target directory: %w", err)
	}
	return len(entries), true, nil
}

func previewPath(targetPath string) (pathInfo, bool, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return pathInfo{}, false, nil
		}
		return pathInfo{}, false, fmt.Errorf("stat preview path: %w", err)
	}
	return pathInfo{
		IsDir:     info.IsDir(),
		SizeBytes: info.Size(),
	}, true, nil
}

func deletePath(targetPath string) (bool, error) {
	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat delete path: %w", err)
	}
	if info.IsDir() {
		if err := os.RemoveAll(targetPath); err != nil {
			return true, fmt.Errorf("remove directory: %w", err)
		}
		return true, nil
	}
	if err := os.Remove(targetPath); err != nil {
		return true, fmt.Errorf("remove file: %w", err)
	}
	return true, nil
}

func guardDeletePath(targetPath string) error {
	cleaned := filepath.Clean(targetPath)
	if cleaned == "/" || cleaned == "." {
		return fmt.Errorf("refusing to delete unsafe path %q", targetPath)
	}
	return nil
}
