package osdetect

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// OSType represents the detected operating system type
type OSType string

const (
	Debian  OSType = "debian"
	Ubuntu  OSType = "ubuntu"
	RHEL    OSType = "rhel"
	CentOS  OSType = "centos"
	Fedora  OSType = "fedora"
	Alpine  OSType = "alpine"
	Unknown OSType = "unknown"
)

// OSInfo contains information about the detected OS
type OSInfo struct {
	Type    OSType
	Name    string
	Version string
	Arch    string
}

// Detect identifies the operating system
func Detect() (*OSInfo, error) {
	info := &OSInfo{
		Type: Unknown,
		Arch: detectArch(),
	}

	// Try /etc/os-release first (most modern systems)
	if osRelease, err := parseOSRelease(); err == nil {
		info.Name = osRelease["NAME"]
		info.Version = osRelease["VERSION_ID"]
		
		// Determine OS type from ID
		id := strings.ToLower(osRelease["ID"])
		switch id {
		case "debian":
			info.Type = Debian
		case "ubuntu":
			info.Type = Ubuntu
		case "rhel", "redhat":
			info.Type = RHEL
		case "centos":
			info.Type = CentOS
		case "fedora":
			info.Type = Fedora
		case "alpine":
			info.Type = Alpine
		default:
			// Check ID_LIKE for derivatives
			if idLike, ok := osRelease["ID_LIKE"]; ok {
				if strings.Contains(idLike, "debian") {
					info.Type = Debian
				} else if strings.Contains(idLike, "rhel") || strings.Contains(idLike, "fedora") {
					info.Type = RHEL
				}
			}
		}
		
		return info, nil
	}

	// Fallback: check specific files
	if fileExists("/etc/debian_version") {
		info.Type = Debian
		info.Name = "Debian"
		if data, err := os.ReadFile("/etc/debian_version"); err == nil {
			info.Version = strings.TrimSpace(string(data))
		}
		return info, nil
	}

	if fileExists("/etc/redhat-release") {
		info.Type = RHEL
		info.Name = "RedHat"
		if data, err := os.ReadFile("/etc/redhat-release"); err == nil {
			info.Version = strings.TrimSpace(string(data))
		}
		return info, nil
	}

	if fileExists("/etc/alpine-release") {
		info.Type = Alpine
		info.Name = "Alpine"
		if data, err := os.ReadFile("/etc/alpine-release"); err == nil {
			info.Version = strings.TrimSpace(string(data))
		}
		return info, nil
	}

	return nil, fmt.Errorf("unable to detect operating system")
}

// parseOSRelease parses /etc/os-release file
func parseOSRelease() (map[string]string, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		// Remove quotes
		value = strings.Trim(value, `"'`)
		
		result[key] = value
	}

	return result, nil
}

// detectArch detects the system architecture
func detectArch() string {
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(output))
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsSupported returns true if the OS is supported
func (i *OSInfo) IsSupported() bool {
	return i.Type != Unknown
}

// String returns a string representation of the OS
func (i *OSInfo) String() string {
	return fmt.Sprintf("%s %s (%s)", i.Name, i.Version, i.Arch)
}
