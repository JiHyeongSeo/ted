package editor

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PythonEnv holds information about a Python environment.
type PythonEnv struct {
	Path    string // path to python executable
	Version string // e.g. "3.12.0"
	VenvName string // e.g. "myenv", ".venv", or "" if system
	IsVenv  bool
}

// DetectPythonEnv detects the current Python environment.
func DetectPythonEnv(projectRoot string) *PythonEnv {
	env := &PythonEnv{}

	// Check VIRTUAL_ENV first
	if venv := os.Getenv("VIRTUAL_ENV"); venv != "" {
		env.Path = filepath.Join(venv, "bin", "python")
		env.VenvName = filepath.Base(venv)
		env.IsVenv = true
		env.Version = getPythonVersion(env.Path)
		return env
	}

	// Check CONDA_DEFAULT_ENV
	if conda := os.Getenv("CONDA_DEFAULT_ENV"); conda != "" && conda != "base" {
		condaPrefix := os.Getenv("CONDA_PREFIX")
		if condaPrefix != "" {
			env.Path = filepath.Join(condaPrefix, "bin", "python")
		} else {
			env.Path = "python"
		}
		env.VenvName = conda
		env.IsVenv = true
		env.Version = getPythonVersion(env.Path)
		return env
	}

	// Check for local venv directories
	venvDirs := []string{".venv", "venv", ".env", "env"}
	for _, dir := range venvDirs {
		venvPath := filepath.Join(projectRoot, dir)
		pythonPath := filepath.Join(venvPath, "bin", "python")
		if _, err := os.Stat(pythonPath); err == nil {
			env.Path = pythonPath
			env.VenvName = dir
			env.IsVenv = true
			env.Version = getPythonVersion(pythonPath)
			return env
		}
	}

	// Fall back to system python
	if path, err := exec.LookPath("python3"); err == nil {
		env.Path = path
		env.Version = getPythonVersion(path)
		return env
	}
	if path, err := exec.LookPath("python"); err == nil {
		env.Path = path
		env.Version = getPythonVersion(path)
		return env
	}

	return nil
}

// ListAvailableVenvs finds virtual environments in and around the project root.
func ListAvailableVenvs(projectRoot string) []PythonEnv {
	var envs []PythonEnv

	// Check local venv directories
	venvDirs := []string{".venv", "venv", ".env", "env"}
	for _, dir := range venvDirs {
		venvPath := filepath.Join(projectRoot, dir)
		pythonPath := filepath.Join(venvPath, "bin", "python")
		if _, err := os.Stat(pythonPath); err == nil {
			envs = append(envs, PythonEnv{
				Path:     pythonPath,
				VenvName: dir,
				IsVenv:   true,
				Version:  getPythonVersion(pythonPath),
			})
		}
	}

	// Check conda envs
	condaEnvsDir := filepath.Join(os.Getenv("HOME"), "miniconda3", "envs")
	if entries, err := os.ReadDir(condaEnvsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			pythonPath := filepath.Join(condaEnvsDir, entry.Name(), "bin", "python")
			if _, err := os.Stat(pythonPath); err == nil {
				envs = append(envs, PythonEnv{
					Path:     pythonPath,
					VenvName: entry.Name() + " (conda)",
					IsVenv:   true,
					Version:  getPythonVersion(pythonPath),
				})
			}
		}
	}

	// Also check ~/.conda/envs
	condaEnvsDir2 := filepath.Join(os.Getenv("HOME"), ".conda", "envs")
	if entries, err := os.ReadDir(condaEnvsDir2); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			pythonPath := filepath.Join(condaEnvsDir2, entry.Name(), "bin", "python")
			if _, err := os.Stat(pythonPath); err == nil {
				envs = append(envs, PythonEnv{
					Path:     pythonPath,
					VenvName: entry.Name() + " (conda)",
					IsVenv:   true,
					Version:  getPythonVersion(pythonPath),
				})
			}
		}
	}

	// System python
	if path, err := exec.LookPath("python3"); err == nil {
		envs = append(envs, PythonEnv{
			Path:    path,
			Version: getPythonVersion(path),
		})
	}

	return envs
}

func getPythonVersion(pythonPath string) string {
	out, err := exec.Command(pythonPath, "--version").CombinedOutput()
	if err != nil {
		return ""
	}
	// "Python 3.12.0\n" -> "3.12.0"
	s := strings.TrimSpace(string(out))
	s = strings.TrimPrefix(s, "Python ")
	return s
}
