package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"appsmanager/core"
)

//go:embed frontend/dist/*
var embeddedDist embed.FS

func main() {
	server := core.NewServer()
	mux := http.NewServeMux()

	server.SetupRoutes(mux)

	// 静态前端文件托管：优先使用本地 frontend/dist，不存在则使用内嵌 FS
	localDist := filepath.Join(".", "frontend", "dist")
	if _, err := os.Stat(localDist); err == nil {
		mux.Handle("/", http.FileServer(http.Dir(localDist)))
	} else {
		subFS, err := fs.Sub(embeddedDist, "frontend/dist")
		if err == nil {
			mux.Handle("/", http.FileServer(http.FS(subFS)))
		} else {
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintf(w, "<h1>AppsManager Engine Running</h1><p>Frontend dist not found.</p>")
			})
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:49999")
	if err != nil {
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			panic(err)
		}
	}

	port := listener.Addr().(*net.TCPAddr).Port
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	fmt.Printf("[AppsManager] 极轻量核心引擎已启动: %s\n", url)

	// 自动拉起默认浏览器窗口
	go func() {
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	}()

	// 优雅终止与安全强杀断路器
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n[AppsManager] 正在安全停止所有服务...")
		var allSvcs []*core.SubService
		for _, p := range server.ConfigMgr.GetProjects() {
			allSvcs = append(allSvcs, p.SubServices...)
		}
		server.ProcMgr.StopAll(allSvcs)
		os.Exit(0)
	}()

	if err := http.Serve(listener, mux); err != nil {
		fmt.Printf("HTTP Server 退出: %v\n", err)
	}
}
