# IntuneWin Packager

A cross-platform Go tool for creating `.intunewin` packages for Microsoft Intune Win32 app deployment.

## Features

- Creates `.intunewin` packages compatible with Microsoft Intune
- Cross-platform (Windows, macOS, Linux)
- No dependencies on Microsoft tools
- AES-256-CBC encryption with HMAC-SHA256 authentication
- Generates proper `Detection.xml` metadata

## Installation

### From Source

```bash
go build -o intunewin-packager .
```

### Pre-built Binaries

Download from the [Releases](../../releases) page.

## Usage

```bash
./intunewin-packager -source <folder> -setup <file> [-output <dir>]
```

### Options

| Flag | Description | Required |
|------|-------------|----------|
| `-source` | Source folder containing the application files | Yes |
| `-setup` | Name of the setup file (e.g., `install.exe`) within the source folder | Yes |
| `-output` | Output directory for the `.intunewin` file (default: current directory) | No |
| `-quiet` | Suppress progress output | No |
| `-version` | Show version information | No |

### Example

```bash
# Package an application
./intunewin-packager -source ./myapp -setup install.exe -output ./output

# Quiet mode (only outputs the path to the created file)
./intunewin-packager -source ./myapp -setup install.exe -quiet
```

## Output Format

The generated `.intunewin` file is a ZIP archive with the following structure:

```
├── IntuneWinPackage/
│   ├── Contents/
│   │   └── IntunePackage.intunewin  (encrypted content)
│   └── Metadata/
│       └── Detection.xml            (encryption metadata)
```

### Detection.xml

Contains metadata required by Intune to decrypt and deploy the application:

- Application name and setup file
- Encryption keys (AES-256, base64 encoded)
- HMAC for integrity verification
- SHA256 hash of original content

## Technical Details

### Encryption

- **Algorithm**: AES-256-CBC with PKCS7 padding
- **Authentication**: HMAC-SHA256
- **Key size**: 256-bit (32 bytes)
- **IV size**: 128-bit (16 bytes)

### Encrypted File Structure

```
[HMAC - 32 bytes][IV - 16 bytes][Encrypted Data]
```

## References

This implementation is based on the reverse-engineered format documented by:

- [Creating IntuneWin files with C#](https://svrooij.io/2023/10/24/create-intunewin-file/) by Stephan van Rooij
- [Decrypting intunewin files](https://svrooij.io/2023/10/09/decrypting-intunewin-files/) by Stephan van Rooij
- [IntuneWin](https://github.com/volodymyrsmirnov/IntuneWin) by Volodymyr Smirnov

## License

This project is licensed under the GNU Affero General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Disclaimer

This is an independent implementation not affiliated with or endorsed by Microsoft. The `.intunewin` format is not officially documented by Microsoft.
