package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/lai323/jt809server"
	"github.com/lai323/jt809server/jt809"
	"github.com/lai323/jt809server/log"
	"github.com/go-kit/log/level"
)

func waitInterrupt(srv *jt809server.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-c
	srv.Shutdown()
}

type Req struct {
	Lon       uint32
	Lat       uint32
	Vec1      uint16
	Vec2      uint16
	Vec3      uint32
	Direction uint16
	Altitude  uint16
}

func main() {
	logger := log.NewLogdevStdoutLogger(level.AllowDebug())
	srv := jt809server.NewServer(logger)

	// 设置链接信息
	srv.UserID = 1
	srv.Password = "1"
	srv.GNSSCenterID = 1
	srv.UpLinkIP = "localhost"
	srv.UpLinkPort = 8085
	srv.DownLinkIP = "localhost"
	srv.DownLinkPort = 8090

	// 链接到监管平台后启动 http 服务接收定位，推送监管平台
	srv.OnConnect = func() {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					level.Error(logger).Log(
						"msg", "simulate panic",
						"error", err,
						"stack", debug.Stack())
				}
			}()
			level.Debug(logger).Log("msg", "start simulate")
			// for {
			// 	simulate(srv, "测A12345", jt809.PlateColorYellow, time.Second*5)
			// }

			http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
				s, err := ioutil.ReadAll(r.Body)
				if err != nil {
					level.Error(logger).Log("msg", "HandleFunc ReadAll", err)
				}
				req := Req{}
				err = json.Unmarshal(s, &req)
				if err != nil {
					level.Error(logger).Log("msg", "HandleFunc Unmarshal", err)
				}
				level.Debug(logger).Log("msg", "HandleFunc", "req", fmt.Sprintf("%#v", req))

				exgmsg := jt809.NewUpExgMsg()
				exgmsg.VehicleNo = jt809.FixedLengthString("测A12345", 21, true)
				exgmsg.VehicleColor = jt809.PlateColorYellow
				loc := jt809.NewUpExgMsgRealLocation()
				loc.Encrypt = 0
				loc.State = &jt809.LocationStatus{ACC: true, Location: true}
				loc.Alarm = &jt809.LocationAlarm{}

				now := time.Now()
				loc.Date = jt809.GNSSDataDate(now)
				loc.Time = jt809.GNSSDataTime(now)
				loc.Lon = req.Lon
				loc.Lat = req.Lat
				loc.Vec1 = req.Vec1 / 10
				loc.Vec2 = req.Vec2 / 10
				loc.Vec3 = req.Vec3 / 10
				loc.Direction = req.Direction
				loc.Altitude = req.Altitude
				exgmsg.SetSubPacket(loc)
				srv.UpRealLocation(exgmsg)
			})

			err := http.ListenAndServe(":80", nil)
			level.Error(logger).Log("msg", "http server", "error", err)
		}()
	}

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
	level.Info(logger).Log("msg", "server exit")
}
