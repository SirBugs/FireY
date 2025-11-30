package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	defaultBaseURL = "https://firestore.googleapis.com/v1/projects/%s/databases/(default)/documents"
	checkInterval  = 30 * time.Minute
	runDuration    = 24 * time.Hour
	statusFile     = ".firey_status.json"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
)

type Config struct {
	ProjectID  string
	CustomURL  string
	PathList   string
	SinglePath string
	Methods    string
	Verbose    bool
	OutputFile string
	KeepAnEye  bool
	Threads    int
	Silence    bool
}

type TestResult struct {
	Timestamp  string `json:"timestamp"`
	URL        string `json:"url"`
	Path       string `json:"path"`
	Method     string `json:"method"`
	StatusCode int    `json:"status_code"`
	Status     string `json:"status"`
	BodyLength int    `json:"body_length"`
	Body       string `json:"body,omitempty"`
}

type StatusInfo struct {
	PID       int       `json:"pid"`
	StartTime time.Time `json:"start_time"`
	NextCheck time.Time `json:"next_check"`
	Iteration int       `json:"iteration"`
}

var asciiArt = `
  _____ _          __   __
 |  ___(_)_ __ ___ \ \ / /
 | |_  | | '__/ _ \ \ V / 
 |  _| | | | |  __/  | |  
 |_|   |_|_|  \___|  |_|  
                          
 Firebase Authorization Tester
 =============================
`

func main() {
	config := parseFlags()

	if !config.Silence {
		fmt.Println(asciiArt)
	}

	if config.ProjectID == "" {
		fmt.Println("Error: Project ID (-i) is required")
		os.Exit(1)
	}

	if config.KeepAnEye {
		if isParentProcess() {
			spawnBackgroundProcess(config)
		} else {
			runKeepAnEyeMode(config)
		}
		return
	}

	paths := getPaths(config)
	methods := getMethods(config)

	if len(paths) == 0 {
		fmt.Println("Error: No paths provided. Use -p or -l")
		os.Exit(1)
	}

	results := runTests(config, paths, methods)
	displayResults(results, config)
	saveResults(results, config)
}

func parseFlags() Config {
	config := Config{}

	flag.StringVar(&config.ProjectID, "i", "", "Project ID (required)")
	flag.StringVar(&config.CustomURL, "u", "", "Custom base URL (optional)")
	flag.StringVar(&config.PathList, "l", "", "File path containing list of paths (one per line)")
	flag.StringVar(&config.SinglePath, "p", "", "Single path to test")
	flag.StringVar(&config.Methods, "m", "", "Comma-separated HTTP methods (default: GET,POST,PATCH,DELETE)")
	flag.BoolVar(&config.Verbose, "v", false, "Verbose output with detailed information")
	flag.StringVar(&config.OutputFile, "o", "", "Output file to save results")
	flag.BoolVar(&config.KeepAnEye, "kae", false, "Keep An Eye mode - run for 24 hours checking every 30 minutes")
	flag.IntVar(&config.Threads, "t", 1, "Number of parallel threads (default: 1)")
	flag.BoolVar(&config.Silence, "s", false, "Silent mode - no banner or extra output")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "FireY - Firebase Authorization Tester\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -i string\n")
		fmt.Fprintf(os.Stderr, "        Project ID (REQUIRED)\n")
		fmt.Fprintf(os.Stderr, "        Example: -i my-firebase-project\n\n")
		fmt.Fprintf(os.Stderr, "  -p string\n")
		fmt.Fprintf(os.Stderr, "        Single path to test\n")
		fmt.Fprintf(os.Stderr, "        Example: -p /users/123\n\n")
		fmt.Fprintf(os.Stderr, "  -l string\n")
		fmt.Fprintf(os.Stderr, "        File containing list of paths (one per line)\n")
		fmt.Fprintf(os.Stderr, "        Example: -l paths.txt\n\n")
		fmt.Fprintf(os.Stderr, "  -m string\n")
		fmt.Fprintf(os.Stderr, "        Comma-separated HTTP methods to test (default: all)\n")
		fmt.Fprintf(os.Stderr, "        Example: -m GET,POST\n\n")
		fmt.Fprintf(os.Stderr, "  -u string\n")
		fmt.Fprintf(os.Stderr, "        Custom base URL (optional, overrides default Firestore URL)\n")
		fmt.Fprintf(os.Stderr, "        Example: -u https://custom-db.googleapis.com/v1/projects/my-project/databases/custom\n\n")
		fmt.Fprintf(os.Stderr, "  -v    Verbose mode - show detailed response information\n\n")
		fmt.Fprintf(os.Stderr, "  -o string\n")
		fmt.Fprintf(os.Stderr, "        Output file to save results\n")
		fmt.Fprintf(os.Stderr, "        Example: -o results.txt\n\n")
		fmt.Fprintf(os.Stderr, "  -kae\n")
		fmt.Fprintf(os.Stderr, "        Keep An Eye mode - run continuously for 24 hours\n")
		fmt.Fprintf(os.Stderr, "        Checks every 30 minutes and saves to output file\n\n")
		fmt.Fprintf(os.Stderr, "  -t int\n")
		fmt.Fprintf(os.Stderr, "        Number of parallel threads (default: 1)\n")
		fmt.Fprintf(os.Stderr, "        Example: -t 5\n\n")
		fmt.Fprintf(os.Stderr, "  -s    Silent mode - no banner or extra information\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s -i my-project -p /users/1\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i my-project -l paths.txt -m GET,POST -v\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i my-project -l paths.txt -kae -o monitoring.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -i my-project -p /admin -t 10 -o results.json\n\n", os.Args[0])
	}

	flag.Parse()
	return config
}

func getPaths(config Config) []string {
	var paths []string

	if config.SinglePath != "" {
		paths = append(paths, config.SinglePath)
	}

	if config.PathList != "" {
		data, err := os.ReadFile(config.PathList)
		if err != nil {
			fmt.Printf("Error reading paths file: %v\n", err)
			os.Exit(1)
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") {
				paths = append(paths, line)
			}
		}
	}

	return paths
}

func getMethods(config Config) []string {
	if config.Methods != "" {
		methods := []string{}
		for _, m := range strings.Split(config.Methods, ",") {
			m = strings.TrimSpace(strings.ToUpper(m))
			if m != "" {
				methods = append(methods, m)
			}
		}
		return methods
	}
	return []string{"GET", "POST", "PATCH", "DELETE"}
}

func buildURL(config Config, path string) string {
	if config.CustomURL != "" {
		return config.CustomURL + path
	}
	baseURL := fmt.Sprintf(defaultBaseURL, config.ProjectID)
	return baseURL + path
}

func getStatusColor(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return colorGreen
	case statusCode == 403:
		return colorYellow
	case statusCode == 401:
		return colorYellow
	case statusCode == 404:
		return colorCyan
	default:
		return colorRed
	}
}

func testEndpoint(url, path, method string) TestResult {
	result := TestResult{
		Timestamp: time.Now().Format(time.RFC3339),
		URL:       url,
		Path:      path,
		Method:    method,
	}

	defer func() {
		if r := recover(); r != nil {
			result.Status = "ERROR"
			result.StatusCode = 0
			result.Body = fmt.Sprintf("Panic: %v", r)
		}
	}()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		result.Status = "ERROR"
		result.Body = err.Error()
		return result
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Status = "ERROR"
		result.Body = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Body = "Failed to read response body"
		result.BodyLength = 0
	} else {
		result.Body = string(body)
		result.BodyLength = len(body)
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		result.Status = "SUCCESS"
	case resp.StatusCode == 403:
		result.Status = "FORBIDDEN"
	case resp.StatusCode == 401:
		result.Status = "UNAUTHORIZED"
	case resp.StatusCode == 404:
		result.Status = "NOT_FOUND"
	default:
		result.Status = "ERROR"
	}

	return result
}

func runTests(config Config, paths []string, methods []string) []TestResult {
	var results []TestResult
	var mutex sync.Mutex
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, config.Threads)

	for _, path := range paths {
		for _, method := range methods {
			wg.Add(1)
			go func(p, m string) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				url := buildURL(config, p)
				result := testEndpoint(url, p, m)

				mutex.Lock()
				results = append(results, result)
				mutex.Unlock()

				if !config.Silence && !config.KeepAnEye {
					color := getStatusColor(result.StatusCode)
					if result.StatusCode >= 200 && result.StatusCode < 300 {
						fmt.Printf("[%s] %s -> %s%d %s (length: %d bytes)%s\n", 
							m, p, 
							color, result.StatusCode, result.Status, result.BodyLength, colorReset)
					} else {
						fmt.Printf("[%s] %s -> %s%d %s%s\n", 
							m, p, 
							color, result.StatusCode, result.Status, colorReset)
					}
				}
			}(path, method)
		}
	}

	wg.Wait()
	return results
}

func displayResults(results []TestResult, config Config) {
	if config.Silence || config.KeepAnEye {
		return
	}

	if config.Verbose {
		fmt.Println("\n" + strings.Repeat("=", 60))
		fmt.Println("Detailed Results")
		fmt.Println(strings.Repeat("=", 60))
		for _, r := range results {
			fmt.Printf("\nTimestamp: %s\n", r.Timestamp)
			fmt.Printf("Path: %s\n", r.Path)
			fmt.Printf("Method: %s\n", r.Method)
			fmt.Printf("Status Code: %d\n", r.StatusCode)
			fmt.Printf("Status: %s\n", r.Status)
			fmt.Printf("Response Length: %d bytes\n", r.BodyLength)
			if r.Body != "" {
				fmt.Printf("Response Body: %s\n", truncate(r.Body, 200))
			}
			fmt.Println(strings.Repeat("-", 60))
		}
	}

	statusCounts := make(map[string]int)
	for _, r := range results {
		key := fmt.Sprintf("%d %s", r.StatusCode, r.Status)
		statusCounts[key]++
	}

	fmt.Println("\n" + strings.Repeat("=", 40))
	fmt.Println("Summary")
	fmt.Println(strings.Repeat("=", 40))
	for status, count := range statusCounts {
		parts := strings.SplitN(status, " ", 2)
		statusCode := 0
		fmt.Sscanf(parts[0], "%d", &statusCode)
		color := getStatusColor(statusCode)
		fmt.Printf("%s%d %s%s: %d\n", color, statusCode, parts[1], colorReset, count)
	}
	fmt.Printf("\nTotal: %d\n", len(results))
}

func saveResults(results []TestResult, config Config) {
	if config.OutputFile == "" {
		return
	}

	file, err := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening output file: %v\n", err)
		return
	}
	defer file.Close()

	for _, r := range results {
		var line string
		if config.Verbose {
			jsonData, _ := json.Marshal(r)
			line = string(jsonData) + "\n"
		} else {
			if r.StatusCode >= 200 && r.StatusCode < 300 {
				line = fmt.Sprintf("[%s] %s %s -> %d %s (length: %d bytes)\n", r.Timestamp, r.Method, r.Path, r.StatusCode, r.Status, r.BodyLength)
			} else {
				line = fmt.Sprintf("[%s] %s %s -> %d %s\n", r.Timestamp, r.Method, r.Path, r.StatusCode, r.Status)
			}
		}
		file.WriteString(line)
	}

	if !config.Silence {
		fmt.Printf("\nResults saved to: %s\n", config.OutputFile)
	}
}

func isParentProcess() bool {
	return os.Getenv("FIREY_BACKGROUND") != "1"
}

func spawnBackgroundProcess(config Config) {
	if config.OutputFile == "" {
		config.OutputFile = "firey_monitoring_" + time.Now().Format("20060102_150405") + ".txt"
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), "FIREY_BACKGROUND=1")
	cmd.Stdout = nil
	cmd.Stderr = nil

	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error starting background process: %v\n", err)
		os.Exit(1)
	}

	status := StatusInfo{
		PID:       cmd.Process.Pid,
		StartTime: time.Now(),
		NextCheck: time.Now().Add(checkInterval),
		Iteration: 0,
	}

	saveStatus(status)

	fmt.Printf("✓ Keep An Eye mode activated!\n")
	fmt.Printf("✓ Background process started (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("✓ Monitoring for: 24 hours\n")
	fmt.Printf("✓ Check interval: 30 minutes\n")
	fmt.Printf("✓ Output file: %s\n", config.OutputFile)
	fmt.Printf("✓ Status file: %s\n\n", statusFile)
	fmt.Printf("To check status: cat %s\n", statusFile)
	fmt.Printf("To stop monitoring: kill %d\n", cmd.Process.Pid)
}

func runKeepAnEyeMode(config Config) {
	paths := getPaths(config)
	methods := getMethods(config)

	if len(paths) == 0 {
		fmt.Println("Error: No paths provided for Keep An Eye mode")
		os.Exit(1)
	}

	startTime := time.Now()
	iteration := 0

	status := StatusInfo{
		PID:       os.Getpid(),
		StartTime: startTime,
		NextCheck: time.Now(),
		Iteration: iteration,
	}
	saveStatus(status)

	for time.Since(startTime) < runDuration {
		iteration++
		
		file, _ := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		file.WriteString(fmt.Sprintf("\n=== Iteration %d - %s ===\n", iteration, time.Now().Format(time.RFC3339)))
		file.Close()

		results := runTests(config, paths, methods)
		saveResults(results, config)

		status.Iteration = iteration
		status.NextCheck = time.Now().Add(checkInterval)
		saveStatus(status)

		if time.Since(startTime)+checkInterval >= runDuration {
			break
		}

		time.Sleep(checkInterval)
	}

	file, _ := os.OpenFile(config.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	file.WriteString(fmt.Sprintf("\n=== Monitoring Completed - %s ===\n", time.Now().Format(time.RFC3339)))
	file.Close()

	os.Remove(statusFile)
}

func saveStatus(status StatusInfo) {
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(statusFile, data, 0644)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}


