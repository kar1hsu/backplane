package storage

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	ErrEmptyFile       = errors.New("upload file is required")
	ErrFileTooLarge    = errors.New("upload file exceeds size limit")
	ErrInvalidFileType = errors.New("upload file type is not allowed")
	ErrInvalidFolder   = errors.New("upload folder is invalid")
)

var folderSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var extensionPattern = regexp.MustCompile(`^\.[A-Za-z0-9]{1,10}$`)

type LocalConfig struct {
	Directory    string
	PublicURL    string
	MaxBytes     int64
	AllowedTypes map[string]string
}

type UploadedFile struct {
	OriginalName string `json:"original_name"`
	FileName     string `json:"file_name"`
	RelativePath string `json:"path"`
	URL          string `json:"url"`
	Size         int64  `json:"size"`
	ContentType  string `json:"content_type"`
}

type Local struct {
	directory    string
	publicURL    string
	maxBytes     int64
	allowedTypes map[string]string
	now          func() time.Time
}

func NewLocal(cfg LocalConfig) (*Local, error) {
	if strings.TrimSpace(cfg.Directory) == "" {
		return nil, errors.New("storage directory is required")
	}
	if cfg.MaxBytes <= 0 {
		return nil, errors.New("storage max bytes must be greater than zero")
	}

	allowedTypes := make(map[string]string, len(cfg.AllowedTypes))
	for extension, contentType := range cfg.AllowedTypes {
		extension = strings.ToLower(strings.TrimSpace(extension))
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		if !extensionPattern.MatchString(extension) {
			return nil, fmt.Errorf("invalid allowed file extension %q", extension)
		}

		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed MIME type %q: %w", contentType, err)
		}
		allowedTypes[extension] = strings.ToLower(mediaType)
	}
	if len(allowedTypes) == 0 {
		return nil, errors.New("storage allowed types must not be empty")
	}

	directory, err := filepath.Abs(cfg.Directory)
	if err != nil {
		return nil, fmt.Errorf("resolve storage directory: %w", err)
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}

	return &Local{
		directory:    directory,
		publicURL:    strings.TrimRight(cfg.PublicURL, "/"),
		maxBytes:     cfg.MaxBytes,
		allowedTypes: allowedTypes,
		now:          time.Now,
	}, nil
}

// Save stores a multipart file under an optional folder. Without a folder it
// uses YYYY/MM/DD based on the application's time.Local.
func (s *Local) Save(file *multipart.FileHeader, folders ...string) (*UploadedFile, error) {
	if file == nil {
		return nil, ErrEmptyFile
	}
	if file.Size > s.maxBytes {
		return nil, ErrFileTooLarge
	}
	if len(folders) > 1 {
		return nil, ErrInvalidFolder
	}

	folder := ""
	if len(folders) == 1 {
		folder = folders[0]
	}
	relativeFolder, err := s.resolveFolder(folder)
	if err != nil {
		return nil, err
	}

	extension := strings.ToLower(filepath.Ext(file.Filename))
	expectedContentType, allowed := s.allowedTypes[extension]
	if !allowed {
		return nil, fmt.Errorf("%w: extension %q", ErrInvalidFileType, extension)
	}

	source, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open upload file: %w", err)
	}
	defer source.Close()

	header := make([]byte, 512)
	headerSize, err := io.ReadFull(source, header)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("read upload file header: %w", err)
	}
	if headerSize == 0 {
		return nil, ErrEmptyFile
	}

	detectedContentType := http.DetectContentType(header[:headerSize])
	mediaType, _, err := mime.ParseMediaType(detectedContentType)
	if err != nil || !strings.EqualFold(mediaType, expectedContentType) {
		return nil, fmt.Errorf(
			"%w: extension %q does not match content type %q",
			ErrInvalidFileType,
			extension,
			detectedContentType,
		)
	}

	targetDirectory := filepath.Join(s.directory, filepath.FromSlash(relativeFolder))
	if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
		return nil, fmt.Errorf("create upload folder: %w", err)
	}

	randomName, err := randomFileName(extension)
	if err != nil {
		return nil, err
	}

	tempFile, err := os.CreateTemp(targetDirectory, ".upload-*")
	if err != nil {
		return nil, fmt.Errorf("create temporary upload file: %w", err)
	}
	tempName := tempFile.Name()
	defer os.Remove(tempName)

	content := io.MultiReader(bytes.NewReader(header[:headerSize]), source)
	written, copyErr := io.Copy(tempFile, io.LimitReader(content, s.maxBytes+1))
	closeErr := tempFile.Close()
	if copyErr != nil {
		return nil, fmt.Errorf("write upload file: %w", copyErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close upload file: %w", closeErr)
	}
	if written > s.maxBytes {
		return nil, ErrFileTooLarge
	}

	if err := os.Chmod(tempName, 0o644); err != nil {
		return nil, fmt.Errorf("set upload file permissions: %w", err)
	}

	targetPath := filepath.Join(targetDirectory, randomName)
	if err := os.Rename(tempName, targetPath); err != nil {
		return nil, fmt.Errorf("save upload file: %w", err)
	}

	relativePath := path.Join(relativeFolder, randomName)
	return &UploadedFile{
		OriginalName: file.Filename,
		FileName:     randomName,
		RelativePath: relativePath,
		URL:          s.publicURL + "/" + relativePath,
		Size:         written,
		ContentType:  mediaType,
	}, nil
}

func (s *Local) resolveFolder(folder string) (string, error) {
	if strings.TrimSpace(folder) == "" {
		return s.now().Format("2006/01/02"), nil
	}

	normalized := strings.ReplaceAll(strings.TrimSpace(folder), `\`, "/")
	if strings.HasPrefix(normalized, "/") {
		return "", ErrInvalidFolder
	}
	normalized = strings.Trim(normalized, "/")
	if normalized == "" || path.Clean(normalized) != normalized {
		return "", ErrInvalidFolder
	}

	for _, segment := range strings.Split(normalized, "/") {
		if !folderSegmentPattern.MatchString(segment) {
			return "", ErrInvalidFolder
		}
	}
	return normalized, nil
}

func randomFileName(extension string) (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate upload file name: %w", err)
	}
	return hex.EncodeToString(value[:]) + extension, nil
}
