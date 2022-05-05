package gateway

import (
	"context"
	"fmt"
	"io"
	"net"

	"next-terminal/server/log"
)

type Tunnel struct {
	ID                string // 唯一标识
	LocalHost         string // 本地监听地址
	LocalPort         int    // 本地端口
	RemoteHost        string // 远程连接地址
	RemotePort        int    // 远程端口
	Gateway           *Gateway
	ctx               context.Context
	cancel            context.CancelFunc
	listener          net.Listener
	localConnections  []net.Conn
	remoteConnections []net.Conn
}

func (r *Tunnel) Open() {
	localAddr := fmt.Sprintf("%s:%d", r.LocalHost, r.LocalPort)

	go func() {
		<-r.ctx.Done()
		_ = r.listener.Close()
		for i := range r.localConnections {
			_ = r.localConnections[i].Close()
		}
		r.localConnections = nil
		for i := range r.remoteConnections {
			_ = r.remoteConnections[i].Close()
		}
		r.remoteConnections = nil
		log.Debugf("SSH 隧道 %v 关闭", localAddr)
	}()
	for {
		log.Debugf("等待客户端访问 %v", localAddr)
		localConn, err := r.listener.Accept()
		if err != nil {
			log.Debugf("接受连接失败 %v, 退出循环", err.Error())
			return
		}
		r.localConnections = append(r.localConnections, localConn)

		log.Debugf("客户端 %v 连接至 %v", localConn.RemoteAddr().String(), localAddr)
		remoteAddr := fmt.Sprintf("%s:%d", r.RemoteHost, r.RemotePort)
		log.Debugf("连接远程主机 %v ...", remoteAddr)
		remoteConn, err := r.Gateway.SshClient.Dial("tcp", remoteAddr)
		if err != nil {
			log.Debugf("连接远程主机 %v 失败", remoteAddr)
			return
		}
		r.remoteConnections = append(r.remoteConnections, remoteConn)

		log.Debugf("连接远程主机 %v 成功", remoteAddr)
		go copyConn(localConn, remoteConn)
		go copyConn(remoteConn, localConn)
		log.Debugf("转发数据 [%v]->[%v]", localAddr, remoteAddr)
	}
}

func (r Tunnel) Close() {
	r.cancel()
}

func copyConn(writer, reader net.Conn) {
	_, _ = io.Copy(writer, reader)
}
