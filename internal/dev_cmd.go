package viz

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type DevServer struct {
	port           uint
	servePort      uint
	serveUrl       string
	enableMemcached bool
	latestError    []byte
	cmd            *exec.Cmd
	memcachedCmd   *exec.Cmd
	mu             sync.Mutex
	lastModTime    map[string]time.Time
}

const memcachedAddr = "localhost:8082"

func DoDev(ctx context.Context, port uint, enableMemcached bool) {
	servePort := port + 1
	serveUrl := "http://localhost:" + strconv.Itoa(int(servePort))

	dev := &DevServer{
		port:           port,
		servePort:      servePort,
		serveUrl:       serveUrl,
		enableMemcached: enableMemcached,
		lastModTime:    make(map[string]time.Time),
	}

	if dev.enableMemcached {
		dev.startMemcached()
	}
	dev.rebuildAndStartServe()

	targetUrl, _ := url.Parse(serveUrl)
	proxy := httputil.NewSingleHostReverseProxy(targetUrl)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if dev.checkForChanges() {
			dev.rebuildAndStartServe()
		}
		if dev.latestError == nil {
			proxy.ServeHTTP(w, r)
		} else {
			w.Write(dev.latestError)
		}
	})

	log.Printf("ready dev http://localhost:%d proxying to %s", port, serveUrl)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(int(port)), nil))
}

func (dev *DevServer) startServe() {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	dev.startServeWithLock()
}

func (dev *DevServer) rebuildAndStartServe() {
	dev.mu.Lock()
	defer dev.mu.Unlock()
	dev.rebuildWithLock()
	dev.startServeWithLock()
}

func (dev *DevServer) startServeWithLock() {
	if dev.cmd != nil {
		dev.cmd.Process.Kill()
		dev.cmd.Wait()
	}

	executable, _ := os.Executable()
	args := []string{"serve", "-port", strconv.Itoa(int(dev.servePort))}
	if dev.enableMemcached {
		args = append(args, "-memcached", memcachedAddr)
	}
	dev.cmd = exec.Command(executable, args...)
	dev.cmd.Stdout = os.Stdout
	dev.cmd.Stderr = os.Stderr

	err := dev.cmd.Start()
	if err != nil {
		log.Printf("Failed to start serve subprocess: %v", err)
		return
	}

	dev.waitForServer()
}

func (dev *DevServer) startMemcached() {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	if dev.memcachedCmd != nil {
		dev.memcachedCmd.Process.Kill()
		dev.memcachedCmd.Wait()
	}

	dev.memcachedCmd = exec.Command("memcached", "-p", "8082", "-v")
	dev.memcachedCmd.Stdout = os.Stdout
	dev.memcachedCmd.Stderr = os.Stderr

	err := dev.memcachedCmd.Start()
	if err != nil {
		log.Printf("Failed to start memcached: %v", err)
		return
	}

	log.Printf("started memcached on port 8082")
}

func (dev *DevServer) rebuildWithLock() {
	start := time.Now()

	var err error

	if err == nil {
		cmd := exec.Command("pnpm", "exec", "esbuild", "js/bundle.ts", "--bundle", "--minify", "--outdir=internal/static", "--log-level=warning", "--sourcemap")
		output, err := cmd.CombinedOutput()
		log.Printf("%s\n", output)
		if err == nil {
			dev.latestError = nil
		} else {
			dev.latestError = output
		}
	}

	if err == nil {
		cmd := exec.Command("pnpm", "exec", "tailwindcss", "--input", "css/main.css", "--output", "internal/static/bundle.css")
		output, err := cmd.CombinedOutput()
		log.Printf("%s\n", output)
		if err == nil {
			dev.latestError = nil
		} else {
			dev.latestError = output
		}
	}

	if err == nil {
		cmd := exec.Command("go", "build", "cmd/viz/viz.go")
		output, err := cmd.CombinedOutput()
		if err == nil {
			dev.latestError = nil
		} else {
			dev.latestError = output
		}
	}

	elapsed := time.Now().Sub(start)

	log.Printf("rebuilt in %6dms\n", elapsed.Milliseconds())
}

func (dev *DevServer) waitForServer() {
	for i := 0; i < 50; i++ {
		resp, err := http.Get(dev.serveUrl)
		if err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("Warning: Server may not be ready")
}

func (dev *DevServer) checkForChanges() bool {
	dev.mu.Lock()
	defer dev.mu.Unlock()

	start := time.Now()
	changed := false

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if filepath.Base(path)[0] == '.' {
			return nil
		}

		modTime := info.ModTime()
		lastMod, exists := dev.lastModTime[path]

		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".html" && ext != ".css" && ext != ".ts" && ext != ".tsx" {
			return nil
		}

		if !exists || modTime.After(lastMod) {
			dev.lastModTime[path] = modTime
			if exists {
				changed = true
			}
		}

		return nil
	})

	elapsed := time.Now().Sub(start)
	log.Printf("checked in %6dms\n", elapsed.Milliseconds())
	return changed
}
