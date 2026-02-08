// Package openpackage provides a Go library for creating .intunewin packages
// for Microsoft Intune Win32 app deployment.
//
// This package can be used as a library or via the CLI tool in cmd/open-package.
//
// Library usage:
//
//	import "github.com/MANCHTOOLS/open-package"
//
//	outputPath, err := openpackage.CreatePackage(openpackage.Options{
//	    SourceDir: "/path/to/app",
//	    SetupFile: "install.exe",
//	    OutputDir: "/path/to/output",
//	})
//
// For more control, you can use the sub-packages directly:
//   - github.com/MANCHTOOLS/open-package/packager - Package creation workflow
//   - github.com/MANCHTOOLS/open-package/crypto - AES-256-CBC encryption
//   - github.com/MANCHTOOLS/open-package/metadata - Detection.xml generation
package openpackage

import (
	"github.com/MANCHTOOLS/open-package/packager"
)

// Options contains the configuration for creating an .intunewin package.
type Options struct {
	// SourceDir is the directory containing the application files
	SourceDir string
	// SetupFile is the name of the setup executable (relative to SourceDir)
	SetupFile string
	// OutputDir is the directory where the .intunewin file will be created
	OutputDir string
	// Quiet suppresses progress output when true
	Quiet bool
}

// CreatePackage creates an .intunewin package from the source directory.
// It returns the path to the created package file.
func CreatePackage(opts Options) (string, error) {
	p := packager.New(packager.Options{
		SourceDir: opts.SourceDir,
		SetupFile: opts.SetupFile,
		OutputDir: opts.OutputDir,
		Quiet:     opts.Quiet,
	})
	return p.CreatePackage()
}

// Packager provides more control over the package creation process.
// Use New to create a Packager instance.
type Packager = packager.Packager

// New creates a new Packager with the given options.
func New(opts Options) *Packager {
	return packager.New(packager.Options{
		SourceDir: opts.SourceDir,
		SetupFile: opts.SetupFile,
		OutputDir: opts.OutputDir,
		Quiet:     opts.Quiet,
	})
}
