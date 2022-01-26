package jt809server

import (
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"github.com/lai323/jt809server/jt809"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

type Server struct {
	UserID       uint32
	Password     string
	GNSSCenterID uint32
	UpLinkIP     string
	UpLinkPort   uint16
	DownLinkIP   string
	DownLinkPort uint16

	upconn   net.Conn
	downconn net.Conn

	logger      log.Logger
	receiveChan chan jt809.Packet
	sngen       *jt809.SerialNoGenerater
	mtx         sync.Mutex
	exitedChan  chan struct{}

	OnConnect func()
}

func NewServer(logger log.Logger) *Server {
	return &Server{
		logger:      logger,
		receiveChan: make(chan jt809.Packet),
		sngen:       jt809.NewSerialNoGenerater(),
		exitedChan:  make(chan struct{}),
	}
}

func (srv *Server) UpRealLocation(loc *jt809.UpExgMsg) {
	srv.send(loc)
}

func (srv *Server) startLinktest(p jt809.Packet) {
	for {
		time.Sleep(time.Second * 50)
		select {
		case <-srv.exitedChan:
			return
		default:
		}
		srv.send(p)
	}
}

func (srv *Server) waitdowconn(ch chan net.Conn) {
	addr := fmt.Sprintf(":%d", srv.DownLinkPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		level.Error(srv.logger).Log("msg", "waitdowconn Listen", "error", err)
		close(ch)
		return
	}

	var tempDelay time.Duration
	for {
		level.Debug(srv.logger).Log("msg", "waitdowconn wait connect", "Addr", ln.Addr())
		conn, err := ln.Accept()

		if err != nil {
			select {
			case <-srv.exitedChan:
				close(ch)
				return
			default:
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				level.Debug(srv.logger).Log(
					"msg", "dowconnConnet listener Accept temporary error",
					"error", err,
					"retrying", tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			level.Error(srv.logger).Log(
				"msg", "dowconnConnet listener Accept error",
				"error", err)
			close(ch)
			return
		}
		ch <- conn
		return
	}
}

func (srv *Server) login() {
	req := jt809.NewUpConnectReq()
	req.UserID = srv.UserID
	req.Password = jt809.FixedLengthString(srv.Password, 8, false)
	req.DownLinkIP = jt809.FixedLengthString(srv.DownLinkIP, 32, false)
	req.DownLinkPort = srv.DownLinkPort
	srv.send(req)
}

func (srv *Server) connect() bool {
	// 等待建立从链路
	ch := make(chan net.Conn)
	safego(func() { srv.waitdowconn(ch) }, srv.logger, "waitdowconn panic")

	// 建立主链路
	addr := fmt.Sprintf("%s:%d", srv.UpLinkIP, srv.UpLinkPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		level.Error(srv.logger).Log("msg", "connect Dial", "error", err)
		return false
	}

	// 启动主链路消息接收
	srv.upconn = conn
	srv.login()
	safego(func() { srv.receive(srv.upconn, "upconn") }, srv.logger, "upconn receive panic")

	// 启动消息处理
	safego(func() { defer srv.Shutdown(); srv.handle() }, srv.logger, "handle panic")

	// 启动从链路消息接收
	var ok bool
	select {
	case conn, ok = <-ch:
		if !ok {
			return false
		}
	case <-srv.exitedChan:
		return false
	}

	srv.downconn = conn
	safego(func() { srv.receive(srv.downconn, "downconn") }, srv.logger, "downconn receive panic")
	return true
}

func (srv *Server) receive(conn net.Conn, connname string) {
	level.Debug(srv.logger).Log("msg", "start receive", "conn", connname)
	dec := jt809.NewDecoder(conn)
	for {
		p, err := dec.Decode()
		if err != nil {
			select {
			case <-srv.exitedChan:
				return
			default:
			}

			if err != io.EOF {
				level.Error(srv.logger).Log("msg", "Server receive Decode error", "error", err)
			}
			break
		}
		level.Debug(srv.logger).Log("msg", "receive", "conn", connname, "packet", p)
		srv.receiveChan <- p
	}
}

func (srv *Server) send(p jt809.Packet) {
	srv.mtx.Lock()
	defer srv.mtx.Unlock()
	lt := p.LinkType()
	var conn net.Conn = nil
	var connname string

	if lt == jt809.DownLinkOnly && srv.downconn == nil {
		level.Error(srv.logger).Log(
			"msg", "Server send downconn unavailable", "packet", p)
		return
	}
	if lt == jt809.UpLinkOnly && srv.upconn == nil {
		level.Error(srv.logger).Log(
			"msg", "Server send upconn unavailable", "packet", p)
		return
	}

	if lt == jt809.DownLinkOnly {
		connname = "downconn"
		conn = srv.downconn
	}
	if lt == jt809.UpLinkOnly {
		connname = "upconn"
		conn = srv.upconn
	}

	if lt == jt809.DownLink {
		connname = "downconn"
		conn = srv.downconn
		if conn == nil {
			connname = "upconn"
			conn = srv.upconn
		}
	}
	if lt == jt809.UpLink {
		connname = "upconn"
		conn = srv.upconn
		if conn == nil {
			connname = "downconn"
			conn = srv.downconn
		}
	}

	if conn == nil {
		level.Error(srv.logger).Log("msg", "Server send not connect available", "packet", p)
	}

	h := p.Header()
	h.SerialNo = srv.sngen.GetByType(h.Type)
	h.GNSSCenterID = srv.GNSSCenterID
	h.Version = []byte{1, 0, 0}
	h.Encrypt = 0
	h.EncryptKey = 0

	err := jt809.NewEncoder(conn).Encode(p)
	if err != nil {
		level.Error(srv.logger).Log("msg", "Server send Encode", "packet", p)
	}
	level.Debug(srv.logger).Log("msg", "send", "conn", connname, "packet", p)
}

func (srv *Server) Serve() error {
	level.Debug(srv.logger).Log("msg", "Server start")

	ok := srv.connect()
	if !ok {
		srv.Shutdown()
		return nil
	}

	if srv.OnConnect != nil {
		func() {
			defer func() {
				if err := recover(); err != nil {
					level.Error(srv.logger).Log(
						"msg", "server OnConnect panic",
						"error", err,
						"stack", debug.Stack())
				}
			}()
			srv.OnConnect()
		}()
	}

	<-srv.exitedChan
	return nil
}

func (srv *Server) Shutdown() {
	select {
	case <-srv.exitedChan:
		// Already closed. Don't close again.
	default:
		close(srv.exitedChan)
		if srv.upconn != nil {
			srv.upconn.Close()
		}
		if srv.downconn != nil {
			srv.downconn.Close()
		}
		close(srv.receiveChan)
	}
}

func (srv *Server) handle() {
	for tmp := range srv.receiveChan {
		handle := func(p jt809.Packet) {
			defer func() {
				if err := recover(); err != nil {

					level.Error(srv.logger).Log(
						"msg", "client handle panic",
						"packet", p,
						"error", err,
						"stack", debug.Stack())
				}
			}()

			header := p.Header()
			switch header.Type {
			case jt809.UP_CONNECT_RSP:
				srv.onUpConnectRsp(p.(*jt809.UpConnectRsp))
			case jt809.DOWN_CONNECT_REQ:
				srv.onDownConnectReq(p.(*jt809.DownConnectReq))
			case jt809.DOWN_LINKTEST_REQ:
				srv.onDownLinkTestReq(p.(*jt809.DownLinkTestReq))
			case jt809.UP_LINKTEST_RSP:
				srv.onUpLinkTestRsp(p.(*jt809.UpLinkTestRsp))
			default:
				level.Info(srv.logger).Log(
					"msg", "Server handle unsupport packet", "packet", p)
			}
		}
		go handle(tmp)
	}
}

func (srv *Server) onUpConnectRsp(p *jt809.UpConnectRsp) {
	level.Info(srv.logger).Log("msg", "login response",
		"Result", p.Result, "VerifyCode", p.VerifyCode)

	// 如果添加了重连逻辑，注意不要启动多个 link test
	testp := jt809.NewUpLinkTestReq()
	safego(func() { srv.startLinktest(testp) }, srv.logger, "upconn linktest panic")
}

func (srv *Server) onDownConnectReq(p *jt809.DownConnectReq) {
	rsp := jt809.NewDownConnectRsp()
	rsp.Result = 0
	srv.send(rsp)
}

func (srv *Server) onDownLinkTestReq(p *jt809.DownLinkTestReq) {
	rsp := jt809.NewDownLinkTestRsp()
	srv.send(rsp)
}

func (srv *Server) onUpLinkTestRsp(p *jt809.UpLinkTestRsp) {
}

func safego(goroutine func(), logger log.Logger, errmsg string) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				level.Error(logger).Log(
					"msg", errmsg,
					"error", err,
					"stack", debug.Stack())
			}
		}()
		goroutine()
	}()
}
