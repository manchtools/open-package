// Package packager implements the .intunewin package creation workflow.
//
// The workflow consists of:
// 1. Create a ZIP archive of the source folder (inner package)
// 2. Encrypt the ZIP using AES-256-CBC with HMAC-SHA256 authentication
// 3. Generate Detection.xml with encryption metadata
// 4. Package everything into the final .intunewin file (outer ZIP)
//
// File structure of the output .intunewin:
//
//	├── IntuneWinPackage/
//	│   ├── Contents/
//	│   │   └── IntunePackage.intunewin (encrypted inner ZIP)
//	│   └── Metadata/
//	│       └── Detection.xml
//
// Based on format documentation from:
// - https://svrooij.io/2023/10/24/create-intunewin-file/
// - https://github.com/volodymyrsmirnov/IntuneWin
package packager

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/MANCHTOOLS/open-package/crypto"
	"github.com/MANCHTOOLS/open-package/metadata"
)

// Options contains the configuration for package creation
type Options struct {
	// SourceDir is the directory containing the application files
	SourceDir string
	// SetupFile is the name of the setup executable (relative to SourceDir)
	SetupFile string
	// OutputDir is the directory where the .intunewin file will be created
	OutputDir string
	// Quiet suppresses progress output
	Quiet bool
}

// Packager handles the creation of .intunewin packages
type Packager struct {
	opts Options
}

// New creates a new Packager with the given options
func New(opts Options) *Packager {
	return &Packager{opts: opts}
}

// log prints a message if not in quiet mode
func (p *Packager) log(format string, args ...interface{}) {
	if !p.opts.Quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// CreatePackage creates the .intunewin package and returns the output path
func (p *Packager) CreatePackage() (string, error) {
	// Step 1: Create inner ZIP of source folder
	p.log("Step 1/4: Creating inner ZIP archive...")
	innerZip, err := p.createInnerZip()
	if err != nil {
		return "", fmt.Errorf("failed to create inner ZIP: %w", err)
	}
	p.log("  Created inner ZIP: %d bytes", len(innerZip))

	// Step 2: Encrypt the inner ZIP
	p.log("Step 2/4: Encrypting content...")
	encInfo, encryptedContent, err := crypto.Encrypt(innerZip)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt content: %w", err)
	}
	p.log("  Encrypted size: %d bytes", len(encryptedContent))

	// Step 3: Generate Detection.xml
	p.log("Step 3/4: Generating Detection.xml...")
	appName := filepath.Base(p.opts.SourceDir)
	detectionXML, err := metadata.GenerateDetectionXML(metadata.DetectionXMLOptions{
		Name:       appName,
		SetupFile:  p.opts.SetupFile,
		CryptoInfo: encInfo.ToBase64(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate Detection.xml: %w", err)
	}

	// Step 4: Create outer ZIP (.intunewin)
	p.log("Step 4/4: Creating .intunewin package...")
	outputPath := filepath.Join(p.opts.OutputDir, appName+".intunewin")
	if err := p.createOuterPackage(outputPath, encryptedContent, detectionXML); err != nil {
		return "", fmt.Errorf("failed to create outer package: %w", err)
	}

	return outputPath, nil
}

// createInnerZip creates a ZIP archive of the source directory
func (p *Packager) createInnerZip() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	baseDir := filepath.Base(p.opts.SourceDir)

	err := filepath.Walk(p.opts.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(p.opts.SourceDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		// Create the archive path (include base directory name)
		archivePath := filepath.Join(baseDir, relPath)
		// Normalize path separators for ZIP format (always use forward slashes)
		archivePath = strings.ReplaceAll(archivePath, string(os.PathSeparator), "/")

		// Create header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create header for %s: %w", relPath, err)
		}
		header.Name = archivePath
		header.Method = zip.Deflate

		if info.IsDir() {
			// Ensure directory entries end with /
			if !strings.HasSuffix(header.Name, "/") {
				header.Name += "/"
			}
			_, err := zw.CreateHeader(header)
			return err
		}

		// Create file entry
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create entry for %s: %w", relPath, err)
		}

		// Copy file content
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", path, err)
		}
		defer file.Close()

		if _, err := io.Copy(writer, file); err != nil {
			return fmt.Errorf("failed to write %s: %w", relPath, err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	return buf.Bytes(), nil
}

// createOuterPackage creates the final .intunewin file with the standard structure
func (p *Packager) createOuterPackage(outputPath string, encryptedContent, detectionXML []byte) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	defer zw.Close()

	// Add Detection.xml to IntuneWinPackage/Metadata/
	metadataPath := "IntuneWinPackage/Metadata/Detection.xml"
	if err := p.addToZip(zw, metadataPath, detectionXML); err != nil {
		return fmt.Errorf("failed to add Detection.xml: %w", err)
	}

	// Add encrypted content to IntuneWinPackage/Contents/
	contentsPath := "IntuneWinPackage/Contents/" + metadata.EncryptedFileName
	if err := p.addToZip(zw, contentsPath, encryptedContent); err != nil {
		return fmt.Errorf("failed to add encrypted content: %w", err)
	}

	return nil
}

// addToZip adds a file to the ZIP archive
func (p *Packager) addToZip(zw *zip.Writer, path string, content []byte) error {
	header := &zip.FileHeader{
		Name:   path,
		Method: zip.Deflate,
	}
	header.SetMode(0644)

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = writer.Write(content)
	return err
}
