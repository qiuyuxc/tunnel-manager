package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"tunnel-manager/handlers"
	"tunnel-manager/setup"
	"tunnel-manager/store"
)

// installedStore opens the configured database for the CLI flags that manage
// an existing administrator account.
func installedStore(storage setup.Storage) *store.Store {
	st := store.NewStore(storage.DSN())
	if !st.Installed() {
		log.Fatalf("面板尚未完成安装：请先启动服务，打开面板完成安装引导（存储配置保存在 %s）", setup.File())
	}
	return st
}

// serveSetup runs the first-run wizard. The panel handlers are never built
// before an administrator account exists, so an uninstalled instance exposes
// nothing but the setup endpoints and the frontend bundle.
func serveSetup(state setup.State, port, staticDir string) {
	wizard := handlers.NewSetupHandler(state, restartProcess)
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Compress(5))
	r.Route("/api", func(r chi.Router) {
		r.Get("/health", healthHandler)
		r.Get("/setup/status", wizard.Status)
		r.Post("/setup/test", wizard.Test)
		r.Post("/setup/complete", wizard.Complete)
	})
	mountStatic(r, staticDir)

	log.Printf("========================================")
	log.Printf("  面板尚未初始化")
	log.Printf("  打开 http://<主机>:%s 按引导选择数据库并创建管理员", port)
	if state.EnvLocked {
		log.Printf("  数据库由环境变量提供，引导只创建管理员账户")
	} else {
		log.Printf("  安装配置将写入 %s", setup.File())
	}
	log.Printf("========================================")
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

// restartProcess re-executes the binary so the freshly installed storage takes
// effect. exec keeps the same process and environment, so no supervisor is
// needed; if it fails the operator restarts the service by hand.
func restartProcess() {
	time.Sleep(300 * time.Millisecond)
	exe, err := os.Executable()
	if err == nil {
		err = syscall.Exec(exe, os.Args, os.Environ())
		if err == nil {
			return
		}
	}
	log.Printf("自动重启失败（%v），请手动重启服务", err)
	os.Exit(0)
}

// mountStatic serves the built frontend, falling back to index.html so the SPA
// owns the client-side routes.
func mountStatic(r chi.Router, staticDir string) {
	if info, err := os.Stat(staticDir); err != nil || !info.IsDir() {
		log.Printf("前端静态目录 %s 不存在，只提供接口", staticDir)
		return
	}
	fs := http.FileServer(http.Dir(staticDir))
	// Font subsets are immutable in practice: a chunk's filename encodes the
	// weight and unicode-range it was generated for, so let clients keep them
	// instead of revalidating a dozen files on every reload. fonts.css keeps
	// revalidating, so re-running scripts/subset-fonts.py still takes effect.
	r.Get("/fonts/*", func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, ".woff2") {
			w.Header().Set("Cache-Control", "public, max-age=2592000")
		}
		fs.ServeHTTP(w, req)
	})
	r.Get("/*", func(w http.ResponseWriter, req *http.Request) {
		// Try to serve the file directly
		path := filepath.Join(staticDir, req.URL.Path)
		if _, err := os.Stat(path); os.IsNotExist(err) || strings.HasSuffix(req.URL.Path, "/") {
			// SPA fallback: serve index.html for missing routes. It must be
			// revalidated on every load: each build deletes the previous
			// hashed bundles, so a client that reuses a cached document ends
			// up requesting assets that no longer exist and renders blank.
			w.Header().Set("Cache-Control", "no-cache")
			http.ServeFile(w, req, filepath.Join(staticDir, "index.html"))
			return
		}
		// Vite puts a content hash in every asset filename, so a given URL can
		// never point at different bytes and clients can keep it indefinitely.
		if strings.HasPrefix(req.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fs.ServeHTTP(w, req)
	})
	log.Printf("Serving static files from %s", staticDir)
}
