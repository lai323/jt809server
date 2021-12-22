package jt809server

import (
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"git.rundle.cn/liuyaqi/jt809server/jt809"
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
}

func NewServer(logger log.Logger) *Server {
	return &Server{
		logger:      logger,
		receiveChan: make(chan jt809.Packet),
		sngen:       jt809.NewSerialNoGenerater(),
		exitedChan:  make(chan struct{}),
	}
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
	level.Debug(srv.logger).Log("msg", "waitdowconn start", "Addr", ln.Addr())

	var tempDelay time.Duration
	for {
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
	safego(func() { srv.receive(conn) }, srv.logger, "upconn receive panic")

	// 启动消息处理
	safego(func() { defer srv.Shutdown(); srv.handle() }, srv.logger, "handle panic")

	// 启动从链路消息接收
	conn, ok := <-ch
	if !ok {
		return false
	}
	srv.downconn = conn
	safego(func() { srv.receive(conn) }, srv.logger, "downconn receive panic")
	return true
}

func (srv *Server) receive(conn net.Conn) {
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
		level.Debug(srv.logger).Log("msg", "Server receive packet", "packet", p)
		srv.receiveChan <- p
	}
}

func (srv *Server) send(p jt809.Packet) {
	srv.mtx.Lock()
	defer srv.mtx.Unlock()
	lt := p.LinkType()
	var conn net.Conn = nil

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
		conn = srv.downconn
	}
	if lt == jt809.UpLinkOnly {
		conn = srv.upconn
	}

	if lt == jt809.DownLink {
		conn = srv.downconn
		if conn == nil {
			conn = srv.upconn
		}
	}
	if lt == jt809.UpLink {
		conn = srv.upconn
		if conn == nil {
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
}

func (srv *Server) Serve() error {
	level.Debug(srv.logger).Log("msg", "Server start")

	ok := srv.connect()
	if !ok {
		srv.Shutdown()
		return nil
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
			default:
				level.Info(srv.logger).Log(
					"msg", "Server handle unsupport packet", "packet", p)
			}
		}
		go handle(tmp)
	}
}

func (srv *Server) onUpConnectRsp(p *jt809.UpConnectRsp) {
	level.Info(srv.logger).Log("msg", "onUpConnectRsp", "packet", p)
	testp := jt809.NewUpLinkTestReq()
	// 如果添加了重连逻辑，注意不要启动多个 link test
	safego(func() { srv.startLinktest(testp) }, srv.logger, "upconn linktest panic")
}

func (srv *Server) onDownConnectReq(p *jt809.DownConnectReq) {
	rsp := jt809.NewDownConnectRsp()
	rsp.Result = 0
	srv.send(rsp)
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
