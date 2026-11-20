package file

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// defaultDirSuggestionLimit caps one typeahead payload; a directory list is a
// hint, not a browsable index, so the first N in walk order are enough.
const defaultDirSuggestionLimit = 500

// SuggestDirs returns workspace-relative directories under the workspace root,
// parents before children and siblings in lexical order, capped at limit
// (<=0 means the default). It applies scan_exclude exactly like a scan: a
// directory whose base name matches a pattern is pruned together with its whole
// subtree, which is what keeps hidden and configured-away directories out of
// the suggestion list. The root itself is never a candidate, and symlinked
// directories are not descended into — the same WalkDir semantics that
// ScanDirectoryOptions uses.
func (s *Service) SuggestDirs(limit int) ([]string, error) {
	if limit <= 0 {
		limit = defaultDirSuggestionLimit
	}

	excludePatterns, _ := s.loadExcludePatterns()
	dirs := make([]string, 0, limit)
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// The root failing to open is the caller's problem; an unreadable
			// subdirectory is skipped, exactly like a scan skips it.
			if path == "." {
				return err
			}
			return nil
		}
		if !d.IsDir() || path == "." {
			return nil
		}
		if matchExclude(excludePatterns, d.Name()) {
			return filepath.SkipDir
		}
		dirs = append(dirs, filepath.ToSlash(path))
		if len(dirs) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}
	return dirs, nil
}
