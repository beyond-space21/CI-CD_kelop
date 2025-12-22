package utils

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"github.com/google/uuid"
)

func SaveVideo(file multipart.File, header *multipart.FileHeader) (string, string, string, error) {
	newVideoID := uuid.New().String()
	splitFilename := strings.Split(header.Filename, ".")
	fileName := newVideoID+ "." + splitFilename[len(splitFilename)-1]
	fullPath := filepath.Join(LocalStorage, "videos", fileName)

	dst, err := os.Create(fullPath)
	if err != nil {
		return "", "", "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", "", "", err
	}

	return fullPath, fileName, newVideoID, nil
}
