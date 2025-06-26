package context

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"supertab/internal/ai"
)

// Collector collects system context information
type Collector struct{}

// NewCollector creates a new context collector
func NewCollector() *Collector {
	return &Collector{}
}

// Collect gathers current system context information
func (c *Collector) Collect() ai.Context {
	ctx := ai.Context{
		DateTime: time.Now(),
		Platform: runtime.GOOS,
	}

	// Get current user
	if user := os.Getenv("USER"); user != "" {
		ctx.User = user
	} else if user := os.Getenv("USERNAME"); user != "" {
		ctx.User = user
	}

	// Get current directory
	if pwd, err := os.Getwd(); err == nil {
		ctx.Directory = pwd
	}

	// Get shell
	if shell := os.Getenv("SHELL"); shell != "" {
		ctx.Shell = shell
	}

	// Get terminal
	if term := os.Getenv("TERM"); term != "" {
		ctx.Terminal = term
	}

	// Check if in git repository and get branch
	c.collectGitInfo(&ctx)

	// Collect system information
	c.collectSystemInfo(&ctx)

	// Collect shell aliases
	c.collectAliases(&ctx)

	// Collect Kubernetes context
	c.collectK8sContext(&ctx)

	return ctx
}

// collectGitInfo checks if current directory is a git repository and gets branch info
func (c *Collector) collectGitInfo(ctx *ai.Context) {
	// Check if .git directory exists
	if _, err := os.Stat(".git"); err == nil {
		ctx.IsGitRepo = true

		// Get current branch
		if cmd := exec.Command("git", "branch", "--show-current"); cmd != nil {
			if output, err := cmd.Output(); err == nil {
				ctx.GitBranch = strings.TrimSpace(string(output))
			}
		}
	}
}

// collectSystemInfo gathers system-specific information
func (c *Collector) collectSystemInfo(ctx *ai.Context) {
	switch runtime.GOOS {
	case "darwin":
		// macOS system info
		if cmd := exec.Command("sw_vers"); cmd != nil {
			if output, err := cmd.Output(); err == nil {
				lines := strings.Split(string(output), "\n")
				var version []string
				for _, line := range lines {
					if strings.Contains(line, ":") {
						parts := strings.SplitN(line, ":", 2)
						if len(parts) == 2 {
							version = append(version, strings.TrimSpace(parts[1]))
						}
					}
				}
				ctx.System = fmt.Sprintf("macOS %s", strings.Join(version, " "))
			}
		}
	case "linux":
		// Linux system info
		if data, err := os.ReadFile("/etc/os-release"); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					name := strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
					ctx.System = name
					break
				}
			}
		}
	case "windows":
		// Windows system info
		ctx.System = "Windows"
	default:
		ctx.System = runtime.GOOS
	}
}

// collectAliases gathers shell aliases from various sources
func (c *Collector) collectAliases(ctx *ai.Context) {
	ctx.Aliases = make(map[string]string)

	// Method 1: Get aliases from current shell session (interactive mode)
	if strings.Contains(ctx.Shell, "zsh") {
		// For zsh, we need to source the configuration and then run alias
		cmd := exec.Command(ctx.Shell, "-i", "-c", "alias")
		cmd.Env = os.Environ()
		if output, err := cmd.Output(); err == nil {
			c.parseAliases(string(output), ctx.Aliases)
		}
	} else if strings.Contains(ctx.Shell, "bash") {
		// For bash, similar approach
		cmd := exec.Command(ctx.Shell, "-i", "-c", "alias")
		cmd.Env = os.Environ()
		if output, err := cmd.Output(); err == nil {
			c.parseAliases(string(output), ctx.Aliases)
		}
	}

	// Method 2: Try sourcing rc files and getting aliases
	if len(ctx.Aliases) == 0 {
		var rcCommand string
		if strings.Contains(ctx.Shell, "zsh") {
			rcCommand = "source ~/.zshrc 2>/dev/null; alias 2>/dev/null"
		} else if strings.Contains(ctx.Shell, "bash") {
			rcCommand = "source ~/.bashrc 2>/dev/null; alias 2>/dev/null"
		}

		if rcCommand != "" {
			cmd := exec.Command("sh", "-c", rcCommand)
			cmd.Env = os.Environ()
			if output, err := cmd.Output(); err == nil {
				c.parseAliases(string(output), ctx.Aliases)
			}
		}
	}

	// Method 3: Read common alias files
	aliasFiles := []string{
		os.Getenv("HOME") + "/.aliases",
		os.Getenv("HOME") + "/.bash_aliases",
		os.Getenv("HOME") + "/.zsh_aliases",
	}

	for _, file := range aliasFiles {
		c.readAliasFile(file, ctx.Aliases)
	}
}

// parseAliases parses alias command output
func (c *Collector) parseAliases(output string, aliases map[string]string) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "alias ") {
			// Parse format: alias name='command'
			parts := strings.SplitN(strings.TrimPrefix(line, "alias "), "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				aliases[name] = value
			}
		} else if strings.Contains(line, "=") && !strings.HasPrefix(line, "-e") {
			// Handle lines that start directly with alias definitions (zsh -i -c alias format)
			// Exclude lines starting with -e (options)
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
				if name != "" && value != "" && !strings.Contains(name, " ") {
					aliases[name] = value
				}
			}
		}
	}
}

// readAliasFile reads aliases from a file
func (c *Collector) readAliasFile(filename string, aliases map[string]string) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "alias ") {
			c.parseAliases(line, aliases)
		}
	}
}

// collectK8sContext gathers Kubernetes environment information
func (c *Collector) collectK8sContext(ctx *ai.Context) {
	k8sCtx := &ai.K8sContext{
		IsAvailable: false,
	}

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		ctx.K8sContext = k8sCtx
		return
	}

	// Check if kubectl can connect to a cluster
	if cmd := exec.Command("kubectl", "config", "current-context"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			k8sCtx.IsAvailable = true
			k8sCtx.CurrentContext = strings.TrimSpace(string(output))
		}
	}

	if !k8sCtx.IsAvailable {
		ctx.K8sContext = k8sCtx
		return
	}

	// Get current namespace
	if cmd := exec.Command("kubectl", "config", "view", "--minify", "--output", "jsonpath={..namespace}"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			namespace := strings.TrimSpace(string(output))
			if namespace == "" {
				namespace = "default"
			}
			k8sCtx.CurrentNamespace = namespace
		}
	}

	// Get cluster info (simplified)
	if cmd := exec.Command("kubectl", "cluster-info", "--request-timeout=2s"); cmd != nil {
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			if len(lines) > 0 {
				k8sCtx.ClusterInfo = strings.TrimSpace(lines[0])
			}
		}
	}

	ctx.K8sContext = k8sCtx
}
