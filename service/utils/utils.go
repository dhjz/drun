package utils

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return strconv.FormatUint(bytes, 10) + " B"
	}
	exp := int(math.Log(float64(bytes)) / math.Log(float64(unit)))
	div := uint64(math.Pow(unit, float64(exp)))
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func Cors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Cache-Control", "max-age=21600")
		return
	}
}

func KillProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return process.Kill()
}

func OpenDir(dir string) error {
	dir = strings.ReplaceAll(dir, "/", "\\")
	cmd := exec.Command("explorer", dir)
	return cmd.Start()
}

func IsProcessRunning(name string) bool {
	cmd := exec.Command("tasklist", "/FI", "IMAGENAME eq "+name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), name)
}

func GetProcessList() ([]string, error) {
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	var processes []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) > 0 {
			processes = append(processes, strings.Trim(parts[0], `"`))
		}
	}
	return processes, nil
}

func KillProcessByName(name string) error {
	cmd := exec.Command("taskkill", "/F", "/IM", name)
	return cmd.Run()
}

func FindPortProcess(port int) (string, int, error) {
	cmd := exec.Command("netstat", "-ano")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(":%d", port)) {
			parts := strings.Fields(line)
			if len(parts) >= 5 && parts[3] == "LISTENING" {
				pid, err := strconv.Atoi(parts[4])
				if err == nil {
					return "process", pid, nil
				}
			}
		}
	}
	return "", 0, fmt.Errorf("未找到占用端口 %d 的进程", port)
}

func KillProcessByPort(port int) error {
	_, pid, err := FindPortProcess(port)
	if err != nil {
		return err
	}
	return KillProcess(pid)
}

func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func NormalizePath(path string) string {
	path = strings.ReplaceAll(path, "/", "\\")
	if strings.HasSuffix(path, "\\") {
		path = path[:len(path)-1]
	}
	return path
}

func GetParentDir(path string) string {
	path = NormalizePath(path)
	lastSep := strings.LastIndex(path, "\\")
	if lastSep == -1 {
		return path
	}
	return path[:lastSep]
}

func GetBaseName(path string) string {
	path = NormalizePath(path)
	lastSep := strings.LastIndex(path, "\\")
	if lastSep == -1 {
		return path
	}
	return path[lastSep+1:]
}

var filepath = struct {
	Walk func(root string, walkFn filepathWalkFunc) error
}{
	Walk: walkDir,
}

type filepathWalkFunc func(path string, info os.FileInfo, err error) error

func walkDir(root string, walkFn filepathWalkFunc) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	return walk(root, info, walkFn)
}

func walk(path string, info os.FileInfo, walkFn filepathWalkFunc) error {
	if !info.IsDir() {
		return walkFn(path, info, nil)
	}

	err := walkFn(path, info, nil)
	if err != nil {
		if !info.IsDir() || err == syscall.ENOENT {
			return err
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return walkFn(path, info, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		path2 := path + "\\" + name
		info2, err := entry.Info()
		if err != nil {
			if err := walkFn(path2, nil, err); err != nil {
				return err
			}
		} else {
			if err := walk(path2, info2, walkFn); err != nil {
				return err
			}
		}
	}
	return nil
}
