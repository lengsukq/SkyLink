package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/skylink/skylink/internal/api"
	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/easytier"
	"github.com/skylink/skylink/internal/proxy"
	"github.com/skylink/skylink/internal/security"
	"github.com/skylink/skylink/internal/store"
	"github.com/skylink/skylink/static"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	configPath := flag.String("config", "", "path to config yaml")
	flag.Parse()

	appCfg, _, easyTierCfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal("config:", err)
	}

	st, err := store.New(appCfg.DBPath)
	if err != nil {
		log.Fatal("store:", err)
	}
	defer st.Close()

	if err := ensureAdminPassword(st); err != nil {
		log.Fatal("admin password:", err)
	}

	pr := proxy.New()
	routes, err := st.HostToBackend()
	if err != nil {
		log.Fatal("load routes:", err)
	}
	pr.SetRoutes(routes)

	// 管理 API + 内嵌静态前端；DDNS 更新器在 API 内按 CF 账号创建客户端并定时更新
	frontendFS, err := fs.Sub(static.FS, "web/dist")
	if err != nil {
		log.Printf("static frontend not available: %v", err)
		frontendFS = nil
	}
	easyTierEnvPath := filepath.Join(filepath.Dir(appCfg.DBPath), "easytier.env")
	daemonManager := easytier.NewDaemonManager()
	runtimeDownloader := easytier.NewRuntimeDownloader(easyTierCfg.RuntimeDir)
	srv := api.New(st, pr, nil, frontendFS, easyTierEnvPath, daemonManager, easyTierCfg, runtimeDownloader)
	adminHandler := srv.Handler()
	defer srv.StopDDNS()

	proxyPort := appCfg.ProxyPort
	if proxyPort <= 0 {
		proxyPort = config.DefaultProxyPort
	}
	adminPort := appCfg.AdminPort
	if adminPort <= 0 {
		adminPort = config.DefaultAdminPort
	}
	go func() {
		log.Printf("proxy listening on :%d", proxyPort)
		if err := http.ListenAndServe(port(proxyPort), pr); err != nil && err != http.ErrServerClosed {
			log.Fatal("proxy server:", err)
		}
	}()

	go func() {
		log.Printf("admin listening on :%d", adminPort)
		if err := http.ListenAndServe(port(adminPort), adminHandler); err != nil && err != http.ErrServerClosed {
			log.Fatal("admin server:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down")
}

func port(p int) string {
	return fmt.Sprintf(":%d", p)
}

func ensureAdminPassword(st *store.Store) error {
	hash, err := st.GetAdminPasswordHash()
	if err != nil {
		return err
	}
	if hash != "" {
		// 已存在密码；不在日志中泄露
		return nil
	}

	pw, err := security.GeneratePassword(18)
	if err != nil {
		return err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := st.SetAdminPasswordHash(string(hashed)); err != nil {
		return err
	}
	log.Printf("SkyLink admin password (generated once): %s", pw)
	return nil
}
