// IntuneWin Packager - A Go implementation for creating .intunewin packages
// This implementation is based on the reverse-engineered format documented by:
// - https://svrooij.io/2023/10/24/create-intunewin-file/
// - https://svrooij.io/2023/10/09/decrypting-intunewin-files/
// - https://github.com/volodymyrsmirnov/IntuneWin
//
// The .intunewin format is an outer ZIP containing:
// - IntuneWinPackage/Metadata/Detection.xml (encryption metadata)
// - IntuneWinPackage/Contents/IntunePackage.intunewin (encrypted inner ZIP)

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MANCHTOOLS/open-package/internal/packager"
)

const (
	version = "1.0.0"
)

func main() {
	// Command line flags
	sourceDir := flag.String("source", "", "Source folder containing the application files (required)")
	setupFile := flag.String("setup", "", "Name of the setup file (e.g., install.exe) within the source folder (required)")
	outputDir := flag.String("output", ".", "Output directory for the .intunewin file")
	showVersion := flag.Bool("version", false, "Show version information")
	quiet := flag.Bool("quiet", false, "Suppress progress output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "IntuneWin Packager v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Creates .intunewin packages for Microsoft Intune Win32 app deployment.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -source <folder> -setup <file> [-output <dir>]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -source ./myapp -setup install.exe -output ./output\n", os.Args[0])
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("IntuneWin Packager v%s\n", version)
		os.Exit(0)
	}

	// Validate required arguments
	if *sourceDir == "" {
		fmt.Fprintln(os.Stderr, "Error: -source is required")
		flag.Usage()
		os.Exit(1)
	}

	if *setupFile == "" {
		fmt.Fprintln(os.Stderr, "Error: -setup is required")
		flag.Usage()
		os.Exit(1)
	}

	// Resolve absolute paths
	absSourceDir, err := filepath.Abs(*sourceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving source path: %v\n", err)
		os.Exit(1)
	}

	absOutputDir, err := filepath.Abs(*outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving output path: %v\n", err)
		os.Exit(1)
	}

	// Verify source directory exists
	info, err := os.Stat(absSourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Source directory does not exist: %s\n", absSourceDir)
		} else {
			fmt.Fprintf(os.Stderr, "Error accessing source directory: %v\n", err)
		}
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: Source path is not a directory: %s\n", absSourceDir)
		os.Exit(1)
	}

	// Verify setup file exists within source directory
	setupPath := filepath.Join(absSourceDir, *setupFile)
	if _, err := os.Stat(setupPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: Setup file not found: %s\n", setupPath)
		} else {
			fmt.Fprintf(os.Stderr, "Error accessing setup file: %v\n", err)
		}
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create the packager
	opts := packager.Options{
		SourceDir: absSourceDir,
		SetupFile: *setupFile,
		OutputDir: absOutputDir,
		Quiet:     *quiet,
	}

	pkg := packager.New(opts)

	if !*quiet {
		fmt.Printf("IntuneWin Packager v%s\n", version)
		fmt.Printf("Source: %s\n", absSourceDir)
		fmt.Printf("Setup file: %s\n", *setupFile)
		fmt.Printf("Output: %s\n", absOutputDir)
		fmt.Println()
	}

	// Create the package
	outputPath, err := pkg.CreatePackage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating package: %v\n", err)
		os.Exit(1)
	}

	if !*quiet {
		fmt.Println()
		fmt.Printf("Successfully created: %s\n", outputPath)
	} else {
		fmt.Println(outputPath)
	}
}
