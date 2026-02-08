package metadata

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/MANCHTOOLS/open-package/crypto"
)

func TestGenerateDetectionXML(t *testing.T) {
	opts := DetectionXMLOptions{
		Name:      "TestApp",
		SetupFile: "install.exe",
		CryptoInfo: crypto.EncryptionInfoBase64{
			EncryptionKey:   "dGVzdGVuY3J5cHRpb25rZXkxMjM0NTY3ODkwMTIzNDU=",
			MacKey:          "dGVzdG1hY2tleWtleTEyMzQ1Njc4OTAxMjM0NTY3ODk=",
			IV:              "dGVzdGl2MTIzNDU2Nzg5MA==",
			MAC:             "dGVzdG1hY3ZhbHVlMTIzNDU2Nzg5MDEyMzQ1Njc4OTA=",
			FileDigest:      "dGVzdGRpZ2VzdDEyMzQ1Njc4OTAxMjM0NTY3ODkwMTI=",
			UnencryptedSize: 1234567,
		},
	}

	xmlData, err := GenerateDetectionXML(opts)
	if err != nil {
		t.Fatalf("GenerateDetectionXML failed: %v", err)
	}

	// Verify XML is well-formed
	var appInfo ApplicationInfo
	if err := xml.Unmarshal(xmlData, &appInfo); err != nil {
		t.Fatalf("Generated XML is not well-formed: %v", err)
	}

	// Verify content
	if appInfo.Name != "TestApp" {
		t.Errorf("Name mismatch: expected TestApp, got %s", appInfo.Name)
	}
	if appInfo.SetupFile != "install.exe" {
		t.Errorf("SetupFile mismatch: expected install.exe, got %s", appInfo.SetupFile)
	}
	if appInfo.FileName != EncryptedFileName {
		t.Errorf("FileName mismatch: expected %s, got %s", EncryptedFileName, appInfo.FileName)
	}
	if appInfo.UnencryptedContentSize != 1234567 {
		t.Errorf("UnencryptedContentSize mismatch: expected 1234567, got %d", appInfo.UnencryptedContentSize)
	}
	if appInfo.ToolVersion != ToolVersion {
		t.Errorf("ToolVersion mismatch: expected %s, got %s", ToolVersion, appInfo.ToolVersion)
	}

	// Verify encryption info
	if appInfo.EncryptionInfo.ProfileIdentifier != ProfileIdentifier {
		t.Errorf("ProfileIdentifier mismatch: expected %s, got %s", ProfileIdentifier, appInfo.EncryptionInfo.ProfileIdentifier)
	}
	if appInfo.EncryptionInfo.FileDigestAlgorithm != FileDigestAlgorithm {
		t.Errorf("FileDigestAlgorithm mismatch: expected %s, got %s", FileDigestAlgorithm, appInfo.EncryptionInfo.FileDigestAlgorithm)
	}
	if appInfo.EncryptionInfo.EncryptionKey != opts.CryptoInfo.EncryptionKey {
		t.Errorf("EncryptionKey mismatch")
	}
	if appInfo.EncryptionInfo.MacKey != opts.CryptoInfo.MacKey {
		t.Errorf("MacKey mismatch")
	}
	if appInfo.EncryptionInfo.InitializationVector != opts.CryptoInfo.IV {
		t.Errorf("InitializationVector mismatch")
	}
	if appInfo.EncryptionInfo.Mac != opts.CryptoInfo.MAC {
		t.Errorf("Mac mismatch")
	}
	if appInfo.EncryptionInfo.FileDigest != opts.CryptoInfo.FileDigest {
		t.Errorf("FileDigest mismatch")
	}

	// Verify XML declaration is present
	xmlString := string(xmlData)
	if !strings.HasPrefix(xmlString, "<?xml") {
		t.Error("XML declaration not present")
	}

	// Verify namespaces are present
	if !strings.Contains(xmlString, "xmlns:xsi") {
		t.Error("xsi namespace not present")
	}
	if !strings.Contains(xmlString, "xmlns:xsd") {
		t.Error("xsd namespace not present")
	}
}
