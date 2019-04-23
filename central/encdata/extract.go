package encdata

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	dataFile = `/stackrox/stackrox-data.tgze`

	readBlockSize = 4096

	targetDir = `/stackrox/data`
)

var (
	log = logging.LoggerForModule()
)

// PrefixExtractedDir prefixes the directory where the extracted + decrypted data is put to
// the (relative) subPath passed in.
func PrefixExtractedDir(subPath string) string {
	return path.Join(targetDir, subPath)
}

func writeDecrypted(inFile *os.File, out io.WriteCloser) error {
	defer utils.IgnoreError(out.Close)

	block, err := aes.NewCipher(key)
	if err != nil {
		return errors.Wrap(err, "creating AES cipher")
	}
	decrypter := cipher.NewCBCDecrypter(block, iv)

	size, err := inFile.Seek(0, 2)
	if err != nil {
		return errors.Wrap(err, "seek error")
	}
	_, err = inFile.Seek(0, 0)
	if err != nil {
		return errors.Wrap(err, "seek error")
	}

	var fileBlock [readBlockSize]byte
	var fileBlockDecrypted [readBlockSize]byte

	totalRead := int64(0)
	lastBlock := false
	for !lastBlock {
		numRead, err := inFile.Read(fileBlock[:])
		if err != nil {
			return errors.Wrap(err, "read error")
		}
		totalRead += int64(numRead)

		currBlock := fileBlock[:numRead]
		currBlockDecrypted := fileBlockDecrypted[:numRead]

		decrypter.CryptBlocks(currBlockDecrypted, currBlock)
		if totalRead == size {
			lastBlock = true
			// Trim PKCS#5 padding
			currBlockDecrypted = currBlockDecrypted[:numRead-int(currBlockDecrypted[numRead-1])]
		}
		_, err = out.Write(currBlockDecrypted)
		if err != nil {
			return errors.Wrap(err, "write error")
		}
	}

	return nil
}

func runCommand(cmd *exec.Cmd, errC chan<- error) {
	_, err := cmd.Output()
	errC <- err
}

func extractEncrypted(ctx context.Context, inputPath string, outputDir string) error {
	inFile, err := os.Open(inputPath)
	if err != nil {
		return err
	}

	tarCmd := exec.Command("tar", "-C", outputDir, "-xzf", "-")
	tarCmd.Stdout = nil
	tarCmd.Stderr = nil

	tarInput, err := tarCmd.StdinPipe()
	if err != nil {
		return err
	}

	errC := make(chan error, 1)
	go runCommand(tarCmd, errC)

	writeErr := writeDecrypted(inFile, tarInput)

	var processErr error
	select {
	case <-ctx.Done():
		processErr = errors.Wrap(ctx.Err(), "context error")
	case processErr = <-errC:
	}

	err = processErr
	if writeErr != nil {
		if err == nil {
			err = errors.Wrap(writeErr, "writing to stdin pipe")
		} else {
			err = errors.Errorf("error running the tar program: %v. Additionally, writing to the pipe failed: %v", err, writeErr)
		}
	}

	return err
}

// ExtractData extracts encrypted stackrox data to /stackrox/data
func ExtractData(ctx context.Context) error {
	markerFile := path.Join(targetDir, ".extracted")
	if _, err := os.Stat(markerFile); err == nil {
		return nil
	}

	if err := extractEncrypted(ctx, dataFile, targetDir); err != nil {
		return err
	}

	f, err := os.Create(markerFile)
	if err != nil {
		log.Errorf("Could not create marker file: %v", err)
	} else {
		_ = f.Close()
	}
	return nil
}

//go:generate ./generate-keys.sh
