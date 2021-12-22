package main

import (
	"os"
	"os/signal"
	"syscall"

	"git.rundle.cn/liuyaqi/jt809server"
	"git.rundle.cn/liuyaqi/jt809server/log"
	"github.com/go-kit/log/level"
)

func waitInterrupt(srv *jt809server.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGTSTP, syscall.SIGQUIT)
	<-c
	srv.Shutdown()
}

func main() {
	logger := log.NewLogdevStdoutLogger(level.AllowDebug())
	srv := jt809server.NewServer(logger)

	srv.UserID = 501001
	srv.Password = "501001"
	srv.GNSSCenterID = 501001
	srv.UpLinkIP = "121.36.37.154"
	srv.UpLinkPort = 8085
	srv.DownLinkIP = "121.89.198.118"
	srv.DownLinkPort = 8090

	done := make(chan struct{})
	go func() {
		err := srv.Serve()
		if err != nil {
			logger.Log("error", err)
			os.Exit(1)
		}
		close(done)
	}()

	waitInterrupt(srv)
	<-done
}
