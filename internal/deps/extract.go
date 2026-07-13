package deps

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/bodgit/sevenzip"
)

// extractMpvExe pulls mpv.exe out of a shinchiro mpv .7z archive.
func extractMpvExe(archivePath, destExe string) error {
	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) == "mpv.exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			out, err := os.Create(destExe)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, rc)
			return err
		}
	}
	return errors.New("mpv.exe no está dentro del archivo .7z")
}
