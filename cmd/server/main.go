package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/skylink/skylink/internal/api"
	"github.com/skylink/skylink/internal/cloudflare"
	"github.com/skylink/skylink/internal/config"
	"github.com/skylink/skylink/internal/ddns"
	"github.com/skylink/skylink/internal/proxy"
	"github.com/skylink/skylink/internal/store"
	"github.com/skylink/skylink/internal/security"
	"github.com/skylink/skylink/static"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	configPath := flag.String("config", "", "path to config yaml")
	flag.Parse()

	appCfg, cfCfg, err := config.Load(*configPath)
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

	var cfClient *cloudflare.Client
	if cfCfg.APIToken != "" {
		cfClient = cloudflare.New(cfCfg.APIToken)
	}

	// DDNS 更新器：更新 CF A 记录并写回 store
	var ddnsUpdater *ddns.Updater
	if cfClient != nil {
		// 目前 DDNS 使用单一 Cloudflare 账号更新 DNS 记录；
		// 如果后续需要按账号拆分，可在此基于 cf_accounts 做进一步扩展。
		ddnsUpdater = ddns.NewUpdater(
			func(item ddns.DDNSItem, newIP string) error {
				_, err := cfClient.UpdateDNSRecord(item.ZoneID, item.RecordID, "A", item.RecordName, newIP, 1, false)
				if err != nil {
					return err
				}
				return st.UpdateDDNSLastResult(item.ID, newIP)
			},
			func() ([]ddns.DDNSItem, error) {
				list, err := st.ListEnabledDDNSConfigs(0)
				if err != nil {
					return nil, err
				}
				out := make([]ddns.DDNSItem, len(list))
				for i := range list {
					out[i] = ddns.DDNSItem{
						ID:          list[i].ID,
						ZoneID:      list[i].ZoneID,
						RecordName:  list[i].RecordName,
						RecordID:    list[i].RecordID,
						IntervalMin: list[i].IntervalMin,
					}
				}
				return out, nil
			},
		)
		ddnsUpdater.Start()
		defer ddnsUpdater.Stop()
	}

	// 管理 API + 内嵌静态前端
	frontendFS, err := fs.Sub(static.FS, "web/dist")
	if err != nil {
		log.Printf("static frontend not available: %v", err)
		frontendFS = nil
	}
	srv := api.New(st, pr, cfClient, frontendFS)
	adminHandler := srv.Handler()

	go func() {
		log.Printf("proxy listening on :%d", appCfg.ProxyPort)
		if err := http.ListenAndServe(port(appCfg.ProxyPort), pr); err != nil && err != http.ErrServerClosed {
			log.Fatal("proxy server:", err)
		}
	}()

	go func() {
		log.Printf("admin listening on :%d", appCfg.AdminPort)
		if err := http.ListenAndServe(port(appCfg.AdminPort), adminHandler); err != nil && err != http.ErrServerClosed {
			log.Fatal("admin server:", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down")
}

func port(p int) string {
	if p <= 0 {
		return fmt.Sprintf(":%d", config.DefaultAdminPort)
	}
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
