package image

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/psanford/memfs"
)

func unzipSource(source string) error {
	// 1. Open the zip file
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer reader.Close()

	// 3. Iterate over zip files inside the archive and unzip each of them
	for _, f := range reader.File {
		err := unzipFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func unzipFile(f *zip.File) error {
	fs := memfs.New()
	//// 4. Check if file paths are not vulnerable to Zip Slip
	//filePath := filepath.Join(destination, f.Name)
	//if !strings.HasPrefix(filePath, filepath.Clean(destination)+string(os.PathSeparator)) {
	//	return fmt.Errorf("invalid file path: %s", filePath)
	//}

	// 5. Create directory tree
	if f.FileInfo().IsDir() {
		if err := fs.MkdirAll(f.Name, os.ModePerm); err != nil {
			return err
		}
		return nil
	}

	if err := fs.MkdirAll(filepath.Dir(f.Name), os.ModePerm); err != nil {
		return err
	}

	// 6. Create a destination file for unzipped content
	destinationFile, err := fs.OpenFile(f.Name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// 7. Unzip the content of a file and copy it to the destination file
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()

	if _, err := io.Copy(destinationFile, zippedFile); err != nil {
		return err
	}
	return nil
}

func DownloadFile(filepath string, url string) error {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
