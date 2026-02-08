// Package metadata handles the Detection.xml generation for .intunewin packages.
//
// The Detection.xml file contains metadata about the encrypted package including:
// - Application name and setup file
// - Encryption keys and parameters (base64 encoded)
// - File hashes for integrity verification
//
// XML Schema based on reverse-engineering documented at:
// - https://svrooij.io/2023/10/24/create-intunewin-file/
// - https://svrooij.io/2023/10/04/analysing-win32-content-prep-tool/
package metadata

import (
	"encoding/xml"
	"fmt"

	"github.com/MANCHTOOLS/open-package/internal/crypto"
)

const (
	// ToolVersion mimics the official Microsoft tool version format
	ToolVersion = "1.8.4.0"
	// ProfileIdentifier is a constant value used in the Detection.xml
	ProfileIdentifier = "ProfileVersion1"
	// FileDigestAlgorithm specifies the hash algorithm used
	FileDigestAlgorithm = "SHA256"
	// EncryptedFileName is the standard name for the encrypted inner package
	EncryptedFileName = "IntunePackage.intunewin"
)

// EncryptionInfo represents the encryption metadata in Detection.xml
type EncryptionInfo struct {
	XMLName             xml.Name `xml:"EncryptionInfo"`
	EncryptionKey       string   `xml:"EncryptionKey"`
	MacKey              string   `xml:"MacKey"`
	InitializationVector string  `xml:"InitializationVector"`
	Mac                 string   `xml:"Mac"`
	ProfileIdentifier   string   `xml:"ProfileIdentifier"`
	FileDigest          string   `xml:"FileDigest"`
	FileDigestAlgorithm string   `xml:"FileDigestAlgorithm"`
}

// ApplicationInfo represents the root element of Detection.xml
type ApplicationInfo struct {
	XMLName              xml.Name       `xml:"ApplicationInfo"`
	XSI                  string         `xml:"xmlns:xsi,attr"`
	XSD                  string         `xml:"xmlns:xsd,attr"`
	ToolVersion          string         `xml:"ToolVersion,attr"`
	Name                 string         `xml:"Name"`
	UnencryptedContentSize int64        `xml:"UnencryptedContentSize"`
	FileName             string         `xml:"FileName"`
	SetupFile            string         `xml:"SetupFile"`
	EncryptionInfo       EncryptionInfo `xml:"EncryptionInfo"`
}

// DetectionXMLOptions contains options for generating Detection.xml
type DetectionXMLOptions struct {
	// Name is the application name (typically the source folder name)
	Name string
	// SetupFile is the name of the setup executable
	SetupFile string
	// EncryptionInfo contains the cryptographic parameters
	CryptoInfo crypto.EncryptionInfoBase64
}

// GenerateDetectionXML creates the Detection.xml content
func GenerateDetectionXML(opts DetectionXMLOptions) ([]byte, error) {
	appInfo := ApplicationInfo{
		XSI:                  "http://www.w3.org/2001/XMLSchema-instance",
		XSD:                  "http://www.w3.org/2001/XMLSchema",
		ToolVersion:          ToolVersion,
		Name:                 opts.Name,
		UnencryptedContentSize: opts.CryptoInfo.UnencryptedSize,
		FileName:             EncryptedFileName,
		SetupFile:            opts.SetupFile,
		EncryptionInfo: EncryptionInfo{
			EncryptionKey:        opts.CryptoInfo.EncryptionKey,
			MacKey:               opts.CryptoInfo.MacKey,
			InitializationVector: opts.CryptoInfo.IV,
			Mac:                  opts.CryptoInfo.MAC,
			ProfileIdentifier:    ProfileIdentifier,
			FileDigest:           opts.CryptoInfo.FileDigest,
			FileDigestAlgorithm:  FileDigestAlgorithm,
		},
	}

	// Generate XML with proper formatting
	xmlData, err := xml.MarshalIndent(appInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Detection.xml: %w", err)
	}

	// Add XML declaration
	xmlHeader := []byte(xml.Header)
	result := append(xmlHeader, xmlData...)

	return result, nil
}
