package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/uyouii/netagent"
	"github.com/uyouii/netagent/base"
	"github.com/uyouii/netagent/common"
	"github.com/uyouii/netagent/tcp_agent"
)

func handleMsg(runningCtx context.Context, agent base.NetAgent) {
	infof, _ := common.GetLogFuns(context.Background())
	for {
		select {
		case netData := <-agent.Receive():
			infof("receive data from client: %v", string(netData.Data))
			respData := fmt.Sprintf("return to client for msg: %v", string(string(netData.Data)))
			agent.SendByConn(runningCtx, netData.ConnInfo, []byte(respData))
		case <-runningCtx.Done():
			infof("done")
			return
		}
	}
}

func sendMsg(runningCtx context.Context, agent base.NetAgent) {
	ctx := context.Background()
	_, errorf := common.GetLogFuns(ctx)
	msgCnt := 0
	for {
		select {
		case <-time.After(time.Second * 5):
			if !agent.Connected() {
				continue
			}
			msgCnt += 1
			err := agent.Send(ctx, []byte(fmt.Sprintf("msg %v sent by server", msgCnt)))
			if err != nil {
				errorf("send msg failed, err: %v", err)
			}
		}
	}
}

func main() {
	infof, errorf := common.GetLogFuns(context.Background())

	cert, err := tls.LoadX509KeyPair("../certs_example/example.crt", "../certs_example/example.key")
	if err != nil {
		errorf("load cert failed, err: %v", err)
		return
	}

	config := &tcp_agent.TcpConfig{
		Addr:      "127.0.0.1",
		Port:      8888,
		ConnCount: 4,
		Mode:      common.AGENT_MODE_SERVER,
		Debug:     true,
		UseTls:    true,
		TlsConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	tcpServer, err := netagent.NewTcpAgent(config, nil, nil, nil, nil)
	if err != nil {
		errorf("create tcp server failed, err : %v")
		return
	}

	err = tcpServer.Start()
	if err != nil {
		errorf("create tcp server failed, err : %v")
		return
	}

	infof("tcp server started, addr: %v, listen port: %v", config.Addr, config.Port)

	runningCtx, cancel := context.WithCancel(context.Background())

	go handleMsg(runningCtx, tcpServer)
	go sendMsg(runningCtx, tcpServer)

	select {
	case <-time.After(time.Second * 60):
		cancel()
	}

	tcpServer.Stop()

	infof("server close")
}
