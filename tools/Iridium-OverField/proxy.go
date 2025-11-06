package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gucooing/lolo/sdk"
)

func newProxy(r *gin.Engine) {
	r.Any("/dispatch/region_info", func(c *gin.Context) {
		resp, err := http.DefaultClient.Post(
			"http://dsp-prod-of.inutan.com:18881/dispatch/region_info",
			"application/x-www-form-urlencoded",
			strings.NewReader(
				"version=2025-09-19-17-04-53"+
					"&version2=4d20f430803c545d5a54de11709da8a8"+
					"&accountType=8888"+
					"&os=2"+
					"&lastloginsdkuid=114514"))
		if err != nil {
			return
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		rsp := new(sdk.RegionInfo)
		err = json.Unmarshal(body, rsp)
		if err != nil {
			return
		}
		ip := rsp.GateTcpIp
		port := rsp.GateTcpPort
		go runTcpProxy(ip, port)
		rsp.GateTcpIp = "10.0.0.4"
		rsp.GateTcpPort = 21001
		c.JSON(http.StatusOK, rsp)
	})
}

var (
	proxy         *Proxy
	remoteAddress string
	stop          chan struct{}
)

func runTcpProxy(ip string, port int) {
	if proxy != nil {
		proxy.Close()
	}
	stop = make(chan struct{})
	remoteAddress = fmt.Sprintf("%s:%d", ip, port)
	listener, err := net.Listen("tcp", ":21001")
	if err != nil {
		return
	}
	defer listener.Close()
	for {
		select {
		case <-stop:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			p, err := NewProxy(conn)
			if err != nil {
				return
			}
			proxy = p
			go p.Run()
		}
	}
}

type Proxy struct {
	localConn  net.Conn
	remoteConn net.Conn
}

func NewProxy(localConn net.Conn) (*Proxy, error) {
	remoteConn, err := net.Dial("tcp", remoteAddress)
	if err != nil {
		return nil, err
	}
	p := &Proxy{
		localConn:  localConn,
		remoteConn: remoteConn,
	}
	// log.Println("new proxy from", p.localConn.RemoteAddr())
	return p, nil
}

func (p *Proxy) Run() {
	defer p.Close()
	done := make(chan struct{}, 2)
	isDone := false
	go func() {
		defer func() {
			if !isDone {
				done <- struct{}{}
			}
		}()
		io.Copy(p.remoteConn, p.localConn)
	}()
	go func() {
		defer func() {
			if !isDone {
				done <- struct{}{}
			}
		}()
		io.Copy(p.localConn, p.remoteConn)
	}()
	if _, ok := <-done; ok {
		isDone = true
		return
	}
}

func (p *Proxy) Close() {
	stop <- struct{}{}
	// log.Println("proxy closed from", p.localConn.RemoteAddr())
	p.localConn.Close()
	p.remoteConn.Close()
	proxy = nil
}
