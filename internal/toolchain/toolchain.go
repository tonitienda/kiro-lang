package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Location struct {
	GoBinary string
	Source   string
}

func Locate() (*Location, error) {
	if explicit := os.Getenv("KIRO_GO_BIN"); explicit != "" {
		if stat, err := os.Stat(explicit); err == nil && !stat.IsDir() {
			return &Location{GoBinary: explicit, Source: "KIRO_GO_BIN"}, nil
		}
		return nil, fmt.Errorf("KIRO_GO_BIN=%q does not point to a usable go binary", explicit)
	}

	for _, dir := range candidateToolchainDirs() {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, "go", "bin", goBinaryName())
		if stat, err := os.Stat(candidate); err == nil && !stat.IsDir() {
			return &Location{GoBinary: candidate, Source: dir}, nil
		}
	}

	pathGo, err := exec.LookPath(goBinaryName())
	if err == nil {
		return &Location{GoBinary: pathGo, Source: "PATH"}, nil
	}
	return nil, fmt.Errorf("unable to find a bundled or system Go toolchain; set KIRO_GO_BIN or place a Go toolchain under toolchain/go relative to the kiro binary")
}

func candidateToolchainDirs() []string {
	var dirs []string
	if envDir := os.Getenv("KIRO_TOOLCHAIN_DIR"); envDir != "" {
		dirs = append(dirs, envDir)
	}
	if exe, err := os.Executable(); err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir := filepath.Dir(exe)
		dirs = append(dirs,
			filepath.Join(exeDir, "toolchain"),
			filepath.Join(exeDir, "..", "toolchain"),
		)
	}
	return dirs
}

func goBinaryName() string {
	if runtime.GOOS == "windows" {
		return "go.exe"
	}
	return "go"
}
