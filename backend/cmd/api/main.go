// API 服务入口：提供 REST API，处理用户登录、配置管理、交易查询。
package main

import (
	"log"
	"time"

	"copyflow/internal/auth"
	"copyflow/internal/bootstrap"
	"copyflow/internal/config"
	"copyflow/internal/handler"
	"copyflow/internal/middleware"
	"copyflow/internal/store"

	"github.com/gin-gonic/gin"
)

// main 启动 HTTP API 服务，注册路由并监听请求。
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	st, err := store.New(cfg.Database.DSN)
	if err != nil {
		log.Fatal(err)
	}
	if err := st.AutoMigrate(); err != nil {
		log.Fatal(err)
	}

	jwt := auth.NewJWTManager(cfg.Auth.JWTSecret, 24*time.Hour)

	authH := handler.NewAuthHandler(st, jwt, "CopyFlow")
	configH := handler.NewConfigHandler(st)
	walletH := handler.NewWalletHandler(st, cfg.Auth.WalletEncryptKey)
	tradeH := handler.NewTradeHandler(st)
	metaH := handler.NewMetaHandler(cfg)

	// 启动时校验链与 DEX 配置是否可用（不阻塞启动）
	if _, err := bootstrap.BuildChains(cfg); err != nil {
		log.Printf("[warn] chain bootstrap: %v", err)
	}
	if _, err := bootstrap.BuildDEXRegistry(cfg); err != nil {
		log.Printf("[warn] dex bootstrap: %v", err)
	}

	gin.SetMode(cfg.Server.Mode)
	r := gin.Default()
	r.Use(middleware.CORS())

	r.GET("/health", metaH.Health)
	r.GET("/api/meta/chains", metaH.Chains)

	api := r.Group("/api")
	{
		api.POST("/auth/nonce", authH.Nonce)
		api.POST("/auth/verify", authH.Verify)

		authed := api.Group("")
		authed.Use(middleware.Auth(jwt))
		{
			authed.GET("/me", authH.Me)
			authed.GET("/configs", configH.List)
			authed.POST("/configs", configH.Create)
			authed.PUT("/configs/:id", configH.Update)
			authed.DELETE("/configs/:id", configH.Delete)
			authed.GET("/wallets", walletH.List)
			authed.POST("/wallets", walletH.Create)
			authed.GET("/copy-trades", tradeH.ListCopyTrades)
			authed.GET("/leader-trades", tradeH.ListLeaderTrades)
		}
	}

	addr := ":" + cfg.Server.Port
	log.Printf("API server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
