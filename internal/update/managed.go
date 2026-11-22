package update

import (
	"os"
	"path/filepath"
	"strings"
)

// Managed returns the package-manager command that owns the binary at exe
// (already passed through filepath.EvalSymlinks), or "" when no package
// manager does. getenv reads SCOOP_GLOBAL and ProgramData.
func Managed(exe, goos string, getenv func(string) string) string {
	if strings.Contains(filepath.ToSlash(exe), "/Cellar/") {
		return "brew upgrade ca"
	}
	if goos != "windows" {
		return ""
	}
	lower := strings.ToLower(strings.ReplaceAll(exe, "/", `\`))
	if !scoopDir(filepath.Dir(exe)) && !strings.Contains(lower, `\scoop\apps\`) {
		return ""
	}
	for _, root := range []string{getenv("SCOOP_GLOBAL"), joinWin(getenv("ProgramData"), "scoop")} {
		if root = strings.ToLower(strings.TrimRight(strings.ReplaceAll(root, "/", `\`), `\`)); root != "" &&
			strings.HasPrefix(lower, root+`\`) {
			return "scoop update -g ca"
		}
	}
	return "scoop update ca"
}

// scoopDir reports whether dir holds Scoop's install record and manifest:
// scoop-install.json and scoop-manifest.json, or install.json and
// manifest.json from before Scoop added the prefix.
func scoopDir(dir string) bool {
	return (exists(filepath.Join(dir, "scoop-install.json")) || exists(filepath.Join(dir, "install.json"))) &&
		(exists(filepath.Join(dir, "scoop-manifest.json")) || exists(filepath.Join(dir, "manifest.json")))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func joinWin(dir, name string) string {
	if dir == "" {
		return ""
	}
	return strings.TrimRight(dir, `\/`) + `\` + name
}
