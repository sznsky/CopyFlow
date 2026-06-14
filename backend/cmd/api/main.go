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
	"copyflow/pkg/email"

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

	mailer := email.NewSender(email.Config{
		Enabled:  cfg.Email.Enabled,
		Host:     cfg.Email.Host,
		Port:     cfg.Email.Port,
		Username: cfg.Email.Username,
		Password: cfg.Email.Password,
		From:     cfg.Email.From,
	})

	authH := handler.NewAuthHandler(st, jwt, "CopyFlow")
	authEmailH := handler.NewAuthEmailHandler(st, jwt, mailer)
	configH := handler.NewConfigHandler(st)
	walletH := handler.NewWalletHandler(st, cfg.Auth.WalletEncryptKey)
	tradeH := handler.NewTradeHandler(st)
	metaH := handler.NewMetaHandler(cfg)

	// 聪明钱处理器（如果启用）
	var smartMoneyH *handler.SmartMoneyHandler
	if cfg.SmartMoney.Enabled && cfg.Dune.Enabled {
		smartMoneyH = handler.NewSmartMoneyHandler(st, cfg.Dune.APIKey, cfg.SmartMoney.ChainID)
		log.Printf("[SmartMoney] Enabled for chain %d", cfg.SmartMoney.ChainID)
	}

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
		api.POST("/auth/email/send-code", authEmailH.SendEmailCode)
		api.POST("/auth/email/register", authEmailH.EmailRegister)
		api.POST("/auth/email/login", authEmailH.EmailLogin)

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

			// 聪明钱 API
			if smartMoneyH != nil {
				authed.GET("/smart-wallets", smartMoneyH.GetTopWallets)
				authed.GET("/token-signals", smartMoneyH.GetTopSignals)
				authed.GET("/token-signals/:id/details", smartMoneyH.GetSignalDetails)
				authed.GET("/wallet-history/:address", smartMoneyH.GetWalletHistory)

				// 管理员接口
				admin := authed.Group("/admin")
				{
					admin.POST("/sync", smartMoneyH.TriggerSync)
					admin.POST("/calculate-scores", smartMoneyH.TriggerScoring)
					admin.POST("/aggregate-signals", smartMoneyH.TriggerSignalAggregation)
				}
			}
		}
	}

	addr := ":" + cfg.Server.Port
	log.Printf("API server listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
