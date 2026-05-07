package router

import (
	"drun/service/utils"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type CommandRequest struct {
	Cmd            string `json:"cmd"`
	Dir            string `json:"dir"`
	WorkDir        string `json:"workDir"`
	Wait           bool   `json:"wait"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	PackageName    string `json:"packageName"`
	PreCommand     string `json:"preCommand"`
	SkipPreCommand bool   `json:"skipPreCommand"`
	SdkPath        string `json:"sdkPath"`
}

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type DataFile struct {
	Projects []ProjectInfo `json:"projects"`
	Tags     []Tag         `json:"tags"`
}

type CommandResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Pid     int    `json:"pid,omitempty"`
}

type ProjectInfo struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Path        string      `json:"path"`
	Commands    []Command   `json:"commands"`
	PackageName string      `json:"packageName"`
	PreCommand  string      `json:"preCommand"`
	SubModules  []SubModule `json:"subModules"`
	CreatedAt   int64       `json:"createdAt"`
	SdkPath     string      `json:"sdkPath"`
	TagIds      []string    `json:"tagIds"`
}

type SubModule struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	JarPath string `json:"jarPath"`
}

type Command struct {
	Name    string `json:"name"`
	Cmd     string `json:"cmd"`
	WorkDir string `json:"workDir"`
	Wait    bool   `json:"wait"`
}

func getDataDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return "data"
	}
	exeDir := filepath.Dir(exePath)
	if exeDir == "" {
		return "data"
	}
	checkFile := filepath.Join(exeDir, "go.mod")
	if _, err := os.Stat(checkFile); err == nil {
		return filepath.Join(exeDir, "data")
	}
	return "data"
}

func SetupRoutesAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello Vue!"))
	})

	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req CommandRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		workDir := req.Dir
		if req.WorkDir != "" {
			workDir = req.WorkDir
		}

		fullCmd := buildCommand(req.Cmd, workDir, req.Type, req.Name, req.PackageName, req.SdkPath)

		if req.PreCommand != "" && !req.SkipPreCommand {
			preCmdStr := strings.TrimSpace(req.PreCommand)
			if strings.Contains(preCmdStr, "git pull") {
				preCmdStr = strings.ReplaceAll(preCmdStr, "git pull", "git -c http.lowSpeedLimit=10 -c http.lowSpeedTime=4 pull")
			}
			preCmd := buildCommand(preCmdStr, req.Dir, req.Type, req.Name, req.PackageName, req.SdkPath)
			fullCmd = preCmd + " && " + fullCmd
			log.Printf("执行命令(含前置): %s", fullCmd)
		} else {
			log.Printf("执行命令: %s, 工作目录: %s", fullCmd, workDir)
		}

		cmd := exec.Command("cmd", "/C", "start", "cmd", "/K", fullCmd)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "LANG=zh_CN.UTF-8")
		if req.SdkPath != "" {
			if req.Type == "backend" {
				cmd.Env = append(cmd.Env, "JAVA_HOME="+req.SdkPath)
				sdkBin := filepath.Join(req.SdkPath, "bin")
				for i, env := range cmd.Env {
					if strings.HasPrefix(env, "PATH=") || strings.HasPrefix(env, "Path=") {
						cmd.Env[i] = env[:5] + sdkBin + ";" + env[5:]
						break
					}
				}
			} else if req.Type == "frontend" {
				for i, env := range cmd.Env {
					if strings.HasPrefix(env, "PATH=") || strings.HasPrefix(env, "Path=") {
						cmd.Env[i] = env[:5] + req.SdkPath + ";" + env[5:]
						break
					}
				}
			}
		}

		err = cmd.Start()
		if err != nil {
			resp := CommandResponse{
				Success: false,
				Message: err.Error(),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		resp := CommandResponse{
			Success: true,
			Pid:     cmd.Process.Pid,
			Message: "命令已启动",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/kill", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req struct {
			Pid int `json:"pid"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		err = utils.KillProcess(req.Pid)
		resp := CommandResponse{
			Success: err == nil,
			Message: "",
		}
		if err != nil {
			resp.Message = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/listDir", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			http.Error(w, "Missing dir parameter", http.StatusBadRequest)
			return
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		type FileInfo struct {
			Name    string `json:"name"`
			IsDir   bool   `json:"isDir"`
			Size    int64  `json:"size"`
			ModTime int64  `json:"modTime"`
		}

		var files []FileInfo
		for _, e := range entries {
			info, _ := e.Info()
			files = append(files, FileInfo{
				Name:    e.Name(),
				IsDir:   e.IsDir(),
				Size:    info.Size(),
				ModTime: info.ModTime().Unix(),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	})

	mux.HandleFunc("/api/browse", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			http.Error(w, "Missing dir parameter", http.StatusBadRequest)
			return
		}

		err := utils.OpenDir(dir)
		resp := CommandResponse{
			Success: err == nil,
			Message: "",
		}
		if err != nil {
			resp.Message = err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/getBatContent", func(w http.ResponseWriter, r *http.Request) {
		batFile := r.URL.Query().Get("file")
		if batFile == "" {
			http.Error(w, "Missing file parameter", http.StatusBadRequest)
			return
		}

		content, err := os.ReadFile(batFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]string{
			"content": string(content),
		})
	})

	mux.HandleFunc("/api/exists", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		if path == "" {
			http.Error(w, "Missing path parameter", http.StatusBadRequest)
			return
		}

		_, err := os.Stat(path)
		exists := err == nil || !os.IsNotExist(err)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"exists": exists})
	})

	mux.HandleFunc("/api/watchProcess", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var req struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		exists := utils.IsProcessRunning(req.Name)
		resp := map[string]bool{"running": exists}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/api/getLog", func(w http.ResponseWriter, r *http.Request) {
		logFile := r.URL.Query().Get("file")
		if logFile == "" {
			http.Error(w, "Missing file parameter", http.StatusBadRequest)
			return
		}

		data, err := os.ReadFile(logFile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		lines := strings.Split(string(data), "\n")
		if len(lines) > 500 {
			lines = lines[len(lines)-500:]
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"content": strings.Join(lines, "\n")})
	})

	mux.HandleFunc("/api/logtail", func(w http.ResponseWriter, r *http.Request) {
		logFile := r.URL.Query().Get("file")
		if logFile == "" {
			http.Error(w, "Missing file parameter", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		lastSize := int64(0)
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				info, err := os.Stat(logFile)
				if err != nil {
					continue
				}

				if info.Size() > lastSize {
					file, err := os.Open(logFile)
					if err != nil {
						continue
					}

					file.Seek(lastSize, 0)
					buf := make([]byte, 4096)
					n, _ := file.Read(buf)
					file.Close()

					if n > 0 {
						content := string(buf[:n])
						content = strings.ReplaceAll(content, "\n", "\n")
						w.Write([]byte("data: " + content + "\n\n"))
						flusher.Flush()
					}

					lastSize = info.Size()
				} else if info.Size() < lastSize {
					lastSize = 0
				}
			}
		}
	})

	mux.HandleFunc("/api/analyzeProject", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		packageName := r.URL.Query().Get("packageName")
		if dir == "" {
			http.Error(w, "Missing dir parameter", http.StatusBadRequest)
			return
		}

		info := analyzeProject(dir, packageName)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	mux.HandleFunc("/api/selectDir", func(w http.ResponseWriter, r *http.Request) {
		psScript := `
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = "选择项目目录"
$dialog.ShowNewFolderButton = $false
if ($dialog.ShowDialog() -eq 'OK') {
	Write-Output $dialog.SelectedPath
}
`
		cmd := exec.Command("powershell", "-Command", psScript)
		output, err := cmd.Output()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"path": ""})
			return
		}
		selectedPath := strings.TrimSpace(string(output))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"path": selectedPath})
	})

	mux.HandleFunc("/api/findJar", func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("dir")
		if dir == "" {
			http.Error(w, "Missing dir parameter", http.StatusBadRequest)
			return
		}

		jarPath := findLatestJar(dir)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"jarPath": jarPath})
	})

	mux.HandleFunc("/api/saveProjects", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		dataDir := getDataDir()
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			http.Error(w, "Failed to create data directory", http.StatusInternalServerError)
			return
		}

		dataFile := filepath.Join(dataDir, "data.json")
		if err := os.WriteFile(dataFile, body, 0644); err != nil {
			http.Error(w, "Failed to save data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
	})

	mux.HandleFunc("/api/loadProjects", func(w http.ResponseWriter, r *http.Request) {
		dataDir := getDataDir()
		dataFile := filepath.Join(dataDir, "data.json")

		data, err := os.ReadFile(dataFile)
		if err != nil {
			oldFile := filepath.Join(dataDir, "projects.json")
			oldData, oldErr := os.ReadFile(oldFile)
			if oldErr == nil {
				var projects []ProjectInfo
				if err := json.Unmarshal(oldData, &projects); err == nil {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{"projects": projects, "tags": []Tag{}})
					return
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"projects": []ProjectInfo{}, "tags": []Tag{}})
			return
		}

		var dataFileContent DataFile
		if err := json.Unmarshal(data, &dataFileContent); err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"projects": []ProjectInfo{}, "tags": []Tag{}})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(dataFileContent)
	})
}

func buildCommand(cmd, workDir, projectType, cmdName string, packageName string, sdkPath string) string {
	cmd = strings.TrimSpace(cmd)

	if strings.HasPrefix(cmd, "start ") || strings.HasPrefix(cmd, "cmd ") {
		return cmd
	}

	if strings.HasPrefix(cmd, "git ") || strings.HasPrefix(cmd, "cd ") || strings.HasPrefix(cmd, "dir ") || strings.HasPrefix(cmd, "echo ") || strings.HasPrefix(cmd, "set ") || strings.HasPrefix(cmd, "call ") {
		return "cd /d " + workDir + " && " + cmd
	}

	origName := strings.ReplaceAll(cmdName, ":", "-")
	pkgname := packageName + "-" + strings.ReplaceAll(cmd, ":", "-")

	switch projectType {
	case "frontend":
		nodePath := "npm"
		if sdkPath != "" {
			nodePath = filepath.Join(sdkPath, "npm.cmd")
		}
		if strings.Contains(origName, "打包") {
			zipName := "dist-" + pkgname + "-" + time.Now().Format("20060102150405") + ".zip"
			return "cd /d " + workDir + " && call " + nodePath + " run " + cmd + " && (if not exist dist-zip mkdir dist-zip) && zip -r dist-zip\\" + zipName + " dist && explorer " + workDir + "\\dist-zip"
		}
		if strings.HasPrefix(cmd, "npm ") || strings.HasPrefix(cmd, "npx ") {
			if sdkPath != "" {
				cmd = strings.Replace(cmd, "npm ", filepath.Join(sdkPath, "npm.cmd")+" ", 1)
				cmd = strings.Replace(cmd, "npx ", filepath.Join(sdkPath, "npx.cmd")+" ", 1)
			}
			return "cd /d " + workDir + " && " + cmd
		}
		return "cd /d " + workDir + " && call " + nodePath + " run " + cmd

	case "backend":
		javaPath := "java"
		if sdkPath != "" {
			javaPath = filepath.Join(sdkPath, "bin", "java.exe")
		}
		if strings.HasPrefix(cmd, "mvn ") || strings.HasPrefix(cmd, "gradle ") {
			return "cd /d " + workDir + " && mvn -v && " + cmd
		}
		if strings.HasPrefix(cmd, "java ") {
			if sdkPath != "" {
				cmd = strings.Replace(cmd, "java ", javaPath+" ", 1)
			}
			return "cd /d " + workDir + " && " + cmd
		}
		return "cd /d " + workDir + " && " + cmd

	case "command":
		if workDir != "" {
			return "cd /d " + workDir + " && " + cmd
		}
		return cmd

	default:
		if workDir != "" {
			return "cd /d " + workDir + " && " + cmd
		}
		return cmd
	}
}

func analyzeProject(dir string, packageName string) ProjectInfo {
	project := ProjectInfo{
		Name:        filepath.Base(dir),
		Type:        "other",
		Path:        dir,
		Commands:    []Command{},
		PackageName: packageName,
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return project
	}

	fileMap := make(map[string]bool)
	for _, f := range files {
		fileMap[strings.ToLower(f.Name())] = true
	}

	switch {
	case fileMap["package.json"]:
		project.Type = "frontend"
		project.Commands = analyzeFrontendProject(dir, fileMap, packageName)

	case fileMap["pom.xml"]:
		project.Type = "backend"
		project.Commands = analyzeMavenProject(dir)
		project.SubModules = analyzeMavenSubModules(dir)

	case fileMap["build.gradle"] || fileMap["build.gradle.kts"]:
		project.Type = "backend"
		project.Commands = analyzeGradleProject(dir)

	case fileMap["go.mod"]:
		project.Type = "backend"
		project.Commands = analyzeGoProject(dir)

	case fileMap["composer.json"]:
		project.Type = "backend"
		project.Commands = analyzePhpProject(dir)

	case fileMap["Cargo.toml"]:
		project.Type = "backend"
		project.Commands = analyzeRustProject(dir)

	default:
		for _, f := range files {
			if f.IsDir() {
				subDir := filepath.Join(dir, f.Name())
				subProject := analyzeProject(subDir, packageName)
				if subProject.Type != "other" {
					project.Type = subProject.Type
					project.Commands = subProject.Commands
					break
				}
			}
		}
	}

	return project
}

func analyzeFrontendProject(dir string, fileMap map[string]bool, packageName string) []Command {
	commands := []Command{}

	commands = append(commands, Command{
		Name:    "安装依赖",
		Cmd:     "npm install",
		WorkDir: dir,
		Wait:    false,
	})

	commands = append(commands, Command{
		Name:    "打包",
		Cmd:     "build",
		WorkDir: dir,
		Wait:    false,
	})

	content, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err == nil {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(content, &pkg) == nil && pkg.Scripts != nil {
			for name := range pkg.Scripts {
				commands = append(commands, Command{
					Name:    name,
					Cmd:     name,
					WorkDir: dir,
					Wait:    false,
				})
			}
		}
	}

	return commands
}

func analyzeMavenProject(dir string) []Command {
	commands := []Command{}

	commands = append(commands, Command{
		Name:    "打包",
		Cmd:     "mvn clean package -DskipTests",
		WorkDir: dir,
		Wait:    false,
	})

	commands = append(commands, Command{
		Name:    "仅编译",
		Cmd:     "mvn compile",
		WorkDir: dir,
		Wait:    false,
	})

	return commands
}

func analyzeMavenSubModules(dir string) []SubModule {
	subModules := []SubModule{}

	pomFile := filepath.Join(dir, "pom.xml")
	data, err := os.ReadFile(pomFile)
	if err != nil {
		log.Printf("读取pom.xml失败: %v", err)
		return subModules
	}

	content := string(data)
	modules := extractMavenModules(content)
	log.Printf("发现 %d 个Maven子模块: %v", len(modules), modules)

	for _, moduleName := range modules {
		modulePath := filepath.Join(dir, moduleName)
		targetDir := filepath.Join(modulePath, "target")

		subModule := SubModule{
			Name: moduleName,
			Path: modulePath,
		}

		files, err := os.ReadDir(targetDir)
		if err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".jar") && !strings.HasSuffix(f.Name(), "-sources.jar") && !strings.HasSuffix(f.Name(), "-javadoc.jar") {
					subModule.JarPath = filepath.Join(targetDir, f.Name())
					break
				}
			}
		}

		subModules = append(subModules, subModule)
	}

	if len(subModules) == 0 {
		targetDir := filepath.Join(dir, "target")
		subModule := SubModule{
			Name: filepath.Base(dir),
			Path: dir,
		}
		files, err := os.ReadDir(targetDir)
		if err == nil {
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".jar") && !strings.HasSuffix(f.Name(), "-sources.jar") && !strings.HasSuffix(f.Name(), "-javadoc.jar") {
					subModule.JarPath = filepath.Join(targetDir, f.Name())
					break
				}
			}
		}
		subModules = append(subModules, subModule)
	}

	return subModules
}

func findLatestJar(dir string) string {
	targetDir := filepath.Join(dir, "target")
	files, err := os.ReadDir(targetDir)
	if err != nil {
		return ""
	}

	var latestJar string
	var latestTime time.Time
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasSuffix(name, ".jar") || strings.HasSuffix(name, "-sources.jar") || strings.HasSuffix(name, "-javadoc.jar") {
			continue
		}
		info, err := f.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			latestJar = filepath.Join(targetDir, name)
		}
	}
	return latestJar
}

func extractMavenModules(content string) []string {
	var modules []string
	lines := strings.Split(content, "\n")
	inModules := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "<modules>") {
			inModules = true
			continue
		}
		if strings.Contains(trimmed, "</modules>") {
			inModules = false
			continue
		}
		if inModules && strings.Contains(trimmed, "<module>") {
			start := strings.Index(trimmed, "<module>") + 8
			end := strings.Index(trimmed, "</module>")
			if start > 7 && end > start {
				moduleName := strings.TrimSpace(trimmed[start:end])
				moduleName = filepath.Clean(moduleName)
				modules = append(modules, moduleName)
			}
		}
	}

	return modules
}

func analyzeGradleProject(dir string) []Command {
	commands := []Command{}

	hasWrapper := false
	if files, _ := os.ReadDir(dir); files != nil {
		for _, f := range files {
			if f.Name() == "gradlew.bat" || f.Name() == "gradlew" {
				hasWrapper = true
				break
			}
		}
	}

	gradleCmd := "gradlew.bat"
	if !hasWrapper {
		gradleCmd = "gradle"
	}

	commands = append(commands, Command{
		Name:    "启动",
		Cmd:     gradleCmd + " bootRun",
		WorkDir: dir,
		Wait:    false,
	})

	commands = append(commands, Command{
		Name:    "打包",
		Cmd:     gradleCmd + " clean build -x test",
		WorkDir: dir,
		Wait:    false,
	})

	return commands
}

func analyzeGoProject(dir string) []Command {
	commands := []Command{}

	commands = append(commands, Command{
		Name:    "运行",
		Cmd:     "go run .",
		WorkDir: dir,
		Wait:    false,
	})

	commands = append(commands, Command{
		Name:    "构建",
		Cmd:     "go build -o app.exe",
		WorkDir: dir,
		Wait:    false,
	})

	commands = append(commands, Command{
		Name:    "下载依赖",
		Cmd:     "go mod tidy",
		WorkDir: dir,
		Wait:    false,
	})

	return commands
}

func analyzePhpProject(dir string) []Command {
	commands := []Command{}

	commands = append(commands, Command{
		Name:    "运行",
		Cmd:     "php -S localhost:8000",
		WorkDir: dir,
		Wait:    false,
	})

	commands = append(commands, Command{
		Name:    "安装依赖",
		Cmd:     "composer install",
		WorkDir: dir,
		Wait:    false,
	})

	return commands
}

func analyzeRustProject(dir string) []Command {
	commands := []Command{}

	commands = append(commands, Command{
		Name:    "运行",
		Cmd:     "cargo run",
		WorkDir: dir,
		Wait:    false,
	})

	commands = append(commands, Command{
		Name:    "构建",
		Cmd:     "cargo build --release",
		WorkDir: dir,
		Wait:    false,
	})

	return commands
}
