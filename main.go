package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/contrib/gzip"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"gucooing/lolo/config"
	"gucooing/lolo/db"
	"gucooing/lolo/gateway"
	"gucooing/lolo/gdconf"
	"gucooing/lolo/logserver"
	"gucooing/lolo/pkg"
	"gucooing/lolo/pkg/alg"
	"gucooing/lolo/pkg/log"
	"gucooing/lolo/sdk"
)

var (
	loloCmd = &cobra.Command{
		Use: pkg.AppName,
		Short: fmt.Sprintf("%s ServerVersion:%s ClientVersion:%s (%s)",
			pkg.AppName, pkg.ServerVersion, pkg.ClientVersion, pkg.Commit),
		Example: fmt.Sprintf("%s --help", pkg.AppName),
		Version: pkg.ServerVersion,
		RunE:    runLolo,
	}

	configCmd = &cobra.Command{
		Use:     "config",
		Aliases: []string{"c"},
		Short:   "配置文件管理",
	}

	genCmd = &cobra.Command{
		Use:   "gen",
		Short: "生成配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.LoadConfig(); err != nil {
				config.SaveConfig(config.DefaultConfig)
			} else {
				config.SaveConfig(config.CONF)
			}
			return nil
		},
	}
)

func init() {
	loloCmd.Flags().StringP("c", "c", "./config.json", "配置文件路径")
	configCmd.AddCommand(genCmd)
	loloCmd.AddCommand(configCmd)

	loloCmd.AddCommand(&cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "版本",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s ServerVersion:%s ClientVersion:%s (%s)\n",
				pkg.AppName, pkg.ServerVersion, pkg.ClientVersion, pkg.Commit)
		},
	})
}

func main() {
	if err := loloCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runLolo(cmd *cobra.Command, args []string) error {
	exit := func() {
		fmt.Printf("\n执行结束请输入任何键退出程序....")
		scanner := bufio.NewScanner(os.Stdin)
		for {
			scanner.Scan()
			return
		}
	}
	fmt.Printf("%s\n", alg.GmNotice)
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("\n程序异常退出,原因:")
			fmt.Println(err)
			exit()
		}
	}()
	if path, err := cmd.Flags().GetString("c"); err == nil && path != "" {
		config.CfgPath = path
	}

	return newLolo()
}

func newLolo() error {
	if err := config.LoadConfig(); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}
	log.NewApp()
	log.App.Info("Lolo Start")
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// 初始化数据库
	log.App.Debug("开始初始化数据库")
	if err := db.NewDB(config.GetDB().GetOption()); err != nil {
		return fmt.Errorf("初始化数据库失败:%s", err.Error())
	}

	log.App.Debug("初始化数据库成功")
	// 初始化资源文件
	gdconf.LoadGameConfig()
	// 初始化gin
	ginRouter, httpServer := NewGin()
	// 初始化sdk
	s := sdk.New(ginRouter)
	// 初始化gateWay
	g := gateway.NewGateway(ginRouter)
	// 初始化logserver
	l := logserver.NewLogServer(ginRouter)

	// 启动HTTP服务器
	log.App.Infof("ClientVersion:%s", pkg.ClientVersion)
	log.App.Infof("ServerVersion:%s", pkg.ServerVersion)
	log.App.Infof("Commit:%s", pkg.Commit)
	go func() {
		log.App.Info("Lolo Http Start!")
		if err := RunGin(httpServer); err != nil {
			if !errors.Is(http.ErrServerClosed, err) {
				log.App.Errorf("HTTP服务器错误:%s", err.Error())
				done <- syscall.SIGTERM
			}
		}
	}()

	// 启动GateWay服务器
	go func() {
		log.App.Info("Lolo GateWay Start!")
		if err := g.RunGateway(); err != nil {
			log.App.Errorf("GateWay服务器错误:%s", err.Error())
			done <- syscall.SIGTERM
		}
	}()

	// 启动LogServer服务器
	go func() {
		log.App.Info("Lolo LogServer Start!")
		if err := l.RunLogServer(); err != nil {
			log.App.Errorf("LogServer服务器错误:%s", err.Error())
			done <- syscall.SIGTERM
		}
	}()
	log.App.Info("Lolo Start!")

	// close
	clo := func() {
		_, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		log.App.Info("Lolo Close...")
		s.Close()
		g.Close()
		l.Close()
		log.Close()
		os.Exit(0)
	}

	go func() {
		select {
		case call := <-done:
			switch call {
			case syscall.SIGINT, syscall.SIGTERM:
				clo()
				return
			}
		}
	}()

	select {}
}

func NewGin() (*gin.Engine, *http.Server) {
	log.App.Debug("初始化gin服务")
	cfg := config.GetHttpNet()
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(
		gin.Recovery(),
		gzip.Gzip(gzip.DefaultCompression),
		log.GinLog(log.App),
	)
	if config.GetMode() == config.ModeDev {
		pprof.Register(router)
	}
	addr := fmt.Sprintf("%s:%s", cfg.GetInnerIp(), cfg.GetInnerPort())
	log.App.Infof("监听地址: http://%s", addr)
	server := &http.Server{Addr: addr, Handler: router}
	log.App.Debug("gin服务初始化成功")
	return router, server
}

func RunGin(server *http.Server) error {
	log.App.Debug("启动http服务")
	return server.ListenAndServe()
}
