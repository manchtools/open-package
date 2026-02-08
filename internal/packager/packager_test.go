package packager

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/MANCHTOOLS/open-package/internal/metadata"
)

func TestCreatePackage(t *testing.T) {
	// Create a temporary source directory
	tempDir, err := os.MkdirTemp("", "intunewin-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "testapp")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create test files
	setupFile := "install.exe"
	setupPath := filepath.Join(sourceDir, setupFile)
	if err := os.WriteFile(setupPath, []byte("fake exe content"), 0644); err != nil {
		t.Fatalf("Failed to create setup file: %v", err)
	}

	// Create a subdirectory with files
	subDir := filepath.Join(sourceDir, "data")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "config.txt"), []byte("config data"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	outputDir := filepath.Join(tempDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Create package
	opts := Options{
		SourceDir: sourceDir,
		SetupFile: setupFile,
		OutputDir: outputDir,
		Quiet:     true,
	}
	pkg := New(opts)

	outputPath, err := pkg.CreatePackage()
	if err != nil {
		t.Fatalf("CreatePackage failed: %v", err)
	}

	// Verify output file exists
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("Output file not found: %v", err)
	}

	// Verify it's a valid ZIP
	zr, err := zip.OpenReader(outputPath)
	if err != nil {
		t.Fatalf("Output is not a valid ZIP: %v", err)
	}
	defer zr.Close()

	// Check expected files exist
	expectedFiles := map[string]bool{
		"IntuneWinPackage/Metadata/Detection.xml":           false,
		"IntuneWinPackage/Contents/IntunePackage.intunewin": false,
	}

	for _, f := range zr.File {
		if _, ok := expectedFiles[f.Name]; ok {
			expectedFiles[f.Name] = true
		}
	}

	for name, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file not found in package: %s", name)
		}
	}

	// Verify Detection.xml content
	for _, f := range zr.File {
		if f.Name == "IntuneWinPackage/Metadata/Detection.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open Detection.xml: %v", err)
			}

			var buf bytes.Buffer
			if _, err := buf.ReadFrom(rc); err != nil {
				rc.Close()
				t.Fatalf("Failed to read Detection.xml: %v", err)
			}
			rc.Close()

			var appInfo metadata.ApplicationInfo
			if err := xml.Unmarshal(buf.Bytes(), &appInfo); err != nil {
				t.Fatalf("Failed to parse Detection.xml: %v", err)
			}

			if appInfo.Name != "testapp" {
				t.Errorf("Name mismatch: expected testapp, got %s", appInfo.Name)
			}
			if appInfo.SetupFile != setupFile {
				t.Errorf("SetupFile mismatch: expected %s, got %s", setupFile, appInfo.SetupFile)
			}
			if appInfo.FileName != "IntunePackage.intunewin" {
				t.Errorf("FileName mismatch: expected IntunePackage.intunewin, got %s", appInfo.FileName)
			}
			if appInfo.UnencryptedContentSize <= 0 {
				t.Error("UnencryptedContentSize should be positive")
			}
			if appInfo.EncryptionInfo.EncryptionKey == "" {
				t.Error("EncryptionKey should not be empty")
			}
			if appInfo.EncryptionInfo.MacKey == "" {
				t.Error("MacKey should not be empty")
			}
			if appInfo.EncryptionInfo.InitializationVector == "" {
				t.Error("InitializationVector should not be empty")
			}
			if appInfo.EncryptionInfo.Mac == "" {
				t.Error("Mac should not be empty")
			}
			if appInfo.EncryptionInfo.FileDigest == "" {
				t.Error("FileDigest should not be empty")
			}
		}
	}
}

func TestCreateInnerZip(t *testing.T) {
	// Create a temporary source directory
	tempDir, err := os.MkdirTemp("", "intunewin-zip-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	sourceDir := filepath.Join(tempDir, "myapp")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create test files
	files := map[string]string{
		"install.exe":     "exe content",
		"readme.txt":      "readme content",
		"data/config.ini": "config content",
	}

	for path, content := range files {
		fullPath := filepath.Join(sourceDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", path, err)
		}
	}

	pkg := New(Options{
		SourceDir: sourceDir,
		SetupFile: "install.exe",
		Quiet:     true,
	})

	zipData, err := pkg.createInnerZip()
	if err != nil {
		t.Fatalf("createInnerZip failed: %v", err)
	}

	// Verify it's a valid ZIP
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("Created data is not a valid ZIP: %v", err)
	}

	// Check all expected files exist (with base dir prefix)
	expectedFiles := map[string]bool{
		"myapp/install.exe":     false,
		"myapp/readme.txt":      false,
		"myapp/data/":           false,
		"myapp/data/config.ini": false,
	}

	for _, f := range zr.File {
		if _, ok := expectedFiles[f.Name]; ok {
			expectedFiles[f.Name] = true
		}
	}

	for name, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file not found in ZIP: %s", name)
		}
	}
}
