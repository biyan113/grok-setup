package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	ConfigName       = "config.toml"
	BackupDirName    = "backups"
	SkillsDirName    = "skills"
	SearchSkillName  = "grok-search"
	OfficialBinName  = "grok"
	SearchConfigName = "config.json"
)

// Home is $GROK_HOME, or ~/.grok (Windows: %USERPROFILE%\.grok).
func Home() (string, error) {
	if v := os.Getenv("GROK_HOME"); v != "" {
		return filepath.Clean(v), nil
	}
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, ".grok"), nil
}

func ConfigFile(home string) string {
	return filepath.Join(home, ConfigName)
}

func BackupDir(home string) string {
	return filepath.Join(home, BackupDirName)
}

func OfficialBin(home string) string {
	return filepath.Join(home, "bin", OfficialBinName)
}

func SearchSkillDir(home string) string {
	return filepath.Join(home, SkillsDirName, SearchSkillName)
}

func SearchConfigFile() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userHome, ".config", "grok-search", SearchConfigName), nil
}

func TTYPath() string {
	if runtime.GOOS == "windows" {
		return "CONIN$"
	}
	return "/dev/tty"
}
