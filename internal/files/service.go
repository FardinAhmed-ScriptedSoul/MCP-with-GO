package files

import (
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type Metadata struct {
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Format  string `json:"format"`
	ModTime string `json:"modified_at"`
}

type CSVData struct {
	Headers   []string   `json:"headers"`
	Rows      [][]string `json:"rows"`
	Truncated bool       `json:"truncated"`
}

const MaxCSVRows = 500

type Service struct {
	roots        []string
	maxFileBytes int64
	maxListFiles int
}

func NewService(roots []string, maxFileBytes int64, maxListFiles int) (*Service, error) {
	if len(roots) == 0 || maxFileBytes <= 0 || maxListFiles <= 0 {
		return nil, fmt.Errorf("file roots and positive size/list limits are required")
	}
	service := &Service{maxFileBytes: maxFileBytes, maxListFiles: maxListFiles}
	for _, root := range roots {
		resolved, err := filepath.EvalSymlinks(root)
		if err != nil {
			return nil, fmt.Errorf("resolve allowed directory %q: %w", root, err)
		}
		info, err := os.Stat(resolved)
		if err != nil {
			return nil, fmt.Errorf("inspect allowed directory %q: %w", root, err)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("allowed path %q is not a directory", root)
		}
		service.roots = append(service.roots, filepath.Clean(resolved))
	}
	return service, nil
}

func (s *Service) ListAllowedFiles() ([]Metadata, error) {
	var files []Metadata
	for _, root := range s.roots {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.IsDir() || !supported(path) {
				return nil
			}
			meta, err := s.metadata(path)
			if err != nil {
				return err
			}
			files = append(files, meta)
			if len(files) > s.maxListFiles {
				return fmt.Errorf("allowed file list exceeds configured limit of %d", s.maxListFiles)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("list files in %q: %w", root, err)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func (s *Service) ReadFile(path string) (string, error) {
	resolved, info, err := s.resolve(path)
	if err != nil {
		return "", err
	}
	if !supported(resolved) {
		return "", fmt.Errorf("unsupported file format %q; supported formats are .txt, .md, and .csv", filepath.Ext(resolved))
	}
	if info.Size() > s.maxFileBytes {
		return "", fmt.Errorf("file exceeds configured size limit of %d bytes", s.maxFileBytes)
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("file is not valid UTF-8 text")
	}
	return string(data), nil
}

func (s *Service) ReadCSV(path string, rowLimit int) (CSVData, error) {
	if rowLimit < 1 || rowLimit > MaxCSVRows {
		return CSVData{}, fmt.Errorf("row limit must be between 1 and %d", MaxCSVRows)
	}
	if !strings.EqualFold(filepath.Ext(path), ".csv") {
		return CSVData{}, fmt.Errorf("CSV path must have a .csv extension")
	}
	content, err := s.ReadFile(path)
	if err != nil {
		return CSVData{}, err
	}
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err != nil {
		return CSVData{}, fmt.Errorf("read CSV header: %w", err)
	}
	if len(headers) == 0 {
		return CSVData{}, fmt.Errorf("CSV must contain at least one column")
	}
	for i, header := range headers {
		headers[i] = strings.TrimSpace(header)
		if headers[i] == "" {
			return CSVData{}, fmt.Errorf("CSV column %d has an empty header", i+1)
		}
	}

	data := CSVData{Headers: headers, Rows: make([][]string, 0)}
	for len(data.Rows) < rowLimit {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				return data, nil
			}
			return CSVData{}, fmt.Errorf("read CSV row %d: %w", len(data.Rows)+2, err)
		}
		if len(row) != len(headers) {
			return CSVData{}, fmt.Errorf("CSV row %d has %d fields; expected %d", len(data.Rows)+2, len(row), len(headers))
		}
		data.Rows = append(data.Rows, row)
	}
	_, err = reader.Read()
	if err == io.EOF {
		return data, nil
	}
	if err != nil {
		return CSVData{}, fmt.Errorf("read CSV row %d: %w", len(data.Rows)+2, err)
	}
	data.Truncated = true
	return data, nil
}

func (s *Service) GetMetadata(path string) (Metadata, error) {
	resolved, _, err := s.resolve(path)
	if err != nil {
		return Metadata{}, err
	}
	if !supported(resolved) {
		return Metadata{}, fmt.Errorf("unsupported file format %q; supported formats are .txt, .md, and .csv", filepath.Ext(resolved))
	}
	return s.metadata(resolved)
}

func (s *Service) metadata(path string) (Metadata, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Metadata{}, fmt.Errorf("inspect file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return Metadata{}, fmt.Errorf("file is not a regular, non-symbolic file")
	}
	return Metadata{
		Path:    path,
		Size:    info.Size(),
		Format:  strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), "."),
		ModTime: info.ModTime().UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *Service) resolve(path string) (string, os.FileInfo, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil, fmt.Errorf("file path is required")
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", nil, fmt.Errorf("resolve file path: %w", err)
	}
	for _, root := range s.roots {
		rel, err := filepath.Rel(root, abs)
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		current := root
		components := strings.Split(rel, string(filepath.Separator))
		for _, component := range components {
			current = filepath.Join(current, component)
			info, err := os.Lstat(current)
			if err != nil {
				return "", nil, fmt.Errorf("inspect file path: %w", err)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return "", nil, fmt.Errorf("symbolic links are not allowed")
			}
		}
		info, err := os.Lstat(abs)
		if err != nil {
			return "", nil, fmt.Errorf("inspect file: %w", err)
		}
		if !info.Mode().IsRegular() {
			return "", nil, fmt.Errorf("path is not a regular file")
		}
		return abs, info, nil
	}
	return "", nil, fmt.Errorf("file path is outside configured directories")
}

func supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".txt", ".md", ".csv":
		return true
	default:
		return false
	}
}
