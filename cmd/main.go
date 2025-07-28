package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/squadracorsepolito/udp-proxy/pkg"
)

func main() {
	ctx, cancelCtx := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer cancelCtx()

	config, err := pkg.LoadConfig()
	if err != nil {
		panic(err)
	}

	for _, proxyCfg := range config.Proxies {
		proxy := pkg.NewProxy(proxyCfg)
		if err := proxy.Init(); err != nil {
			panic(err)
		}
		go proxy.Run(ctx)
	}

	<-ctx.Done()
}
