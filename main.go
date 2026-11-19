package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bing-search-api/internal/api"
	"bing-search-api/internal/bing"
)

func main() {
	addr := flag.String("addr", ":8080", "监听地址，例如 :8080")
	timeout := flag.Duration("timeout", 20*time.Second, "请求 Bing 的超时时间")
	rate := flag.Int("rate", 20, "每客户端每秒允许的请求数（0 表示不限流，但仍保留最低速率）")
	flag.Parse()

	client := bing.NewClient(*timeout)
	api.SetRequestTimeout(*timeout + 5*time.Second)
	handler := api.NewServer(client, *rate)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("Bing Search API 已启动: http://%s", *addr)
		log.Printf("接口: /search/web /search/images /search/videos /search/news /search/all /suggest /health")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("收到退出信号，正在关闭服务...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("关闭服务时出错: %v", err)
	}
	log.Println("服务已退出")
}
