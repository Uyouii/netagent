package main

import (
	"context"
	"fmt"
	"time"

	"github.com/uyouii/netagent"
	"github.com/uyouii/netagent/base"
	"github.com/uyouii/netagent/common"
	"github.com/uyouii/netagent/tcp_agent"
)

func handleMsg(runningCtx context.Context, receiveChan chan *base.RecvNetData) {
	infof, _ := common.GetLogFuns(context.Background())
	for {
		select {
		case netData := <-receiveChan:
			infof("receive data from server: %v", string(netData.Data))
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
			err := agent.Send(ctx, []byte(fmt.Sprintf("msg %v sent by client", msgCnt)))
			if err != nil {
				errorf("send msg failed, err: %v", err)
			}
		}
	}
}

func main() {
	infof, errorf := common.GetLogFuns(context.Background())
	config := &tcp_agent.TcpConfig{
		Addr:      "127.0.0.1",
		Port:      8888,
		ConnCount: 4,
		Mode:      common.AGENT_MODE_CLIENT,
		Debug:     true,
	}
	tcpClient, err := netagent.NewTcpAgent(config, nil, nil, nil, nil)
	if err != nil {
		errorf("create tcp client failed, err : %v")
		return
	}

	err = tcpClient.Start()
	if err != nil {
		errorf("create tcp client failed, err : %v")
		return
	}

	runningCtx, cancel := context.WithCancel(context.Background())

	go handleMsg(runningCtx, tcpClient.Receive())
	go sendMsg(runningCtx, tcpClient)

	select {
	case <-time.After(time.Second * 30):
		cancel()
	}

	tcpClient.Stop()

	infof("client close")
}
