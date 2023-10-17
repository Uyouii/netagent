package tcp_agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/uyouii/netagent/base"
	"github.com/uyouii/netagent/common"
)

type ConnInfo struct {
	conn net.Conn
	id   int
}

type TcpAgent struct {
	recvChan chan *base.RecvNetData
	connChan chan bool

	connMu         sync.Mutex
	curConnCount   int
	maxConnCount   int
	conns          map[int]*ConnInfo // connId to connInfo
	nextConnId     int
	lastSendConnId int

	encoder EncoderFunc
	decoder DecoderFunc

	getInfof  common.GetLogfFunc
	getErrorf common.GetLogfFunc

	mode common.AgentMode
	conf TcpConfig

	runningCtx context.Context
	stop       context.CancelFunc

	runningStat base.RuningStat

	debug bool
}

func NewTcpAgent(tcpConf *TcpConfig, encoder EncoderFunc, decoder DecoderFunc,
	getInfof common.GetLogfFunc, getErrorf common.GetLogfFunc) (base.NetAgent, error) {

	if tcpConf.ConnCount <= 0 || (tcpConf.Mode != common.AGENT_MODE_SERVER && tcpConf.Mode != common.AGENT_MODE_CLIENT) {
		return nil, common.GetError(common.ERROR_INVALID_PARAMS)
	}
	if encoder == nil {
		encoder = DefaultTcpEncoder
	}
	if decoder == nil {
		decoder = DefaultTcpDecoder
	}

	if getInfof == nil {
		getInfof = common.GetInfof
	}

	if getErrorf == nil {
		getErrorf = common.GetErrorf
	}

	agent := TcpAgent{
		recvChan:       make(chan *base.RecvNetData, 64),
		connChan:       make(chan bool, tcpConf.ConnCount*2),
		maxConnCount:   tcpConf.ConnCount,
		curConnCount:   0,
		conf:           *tcpConf,
		encoder:        encoder,
		decoder:        decoder,
		nextConnId:     1,
		lastSendConnId: 0,
		debug:          tcpConf.Debug,
		getInfof:       getInfof,
		getErrorf:      getErrorf,
		conns:          make(map[int]*ConnInfo),
		mode:           tcpConf.Mode,
	}
	return &agent, nil
}

func (agent *TcpAgent) Connected() bool {
	return agent.curConnCount > 0
}

func (agent *TcpAgent) ConnectedCnt() int {
	return agent.curConnCount
}

func (agent *TcpAgent) Start() error {
	ctx := context.Background()
	infof, _ := agent.getInfof(ctx), agent.getErrorf(ctx)

	infof("tcp agent starting")

	agent.resetRunningStat()

	agent.runningCtx, agent.stop = context.WithCancel(context.Background())
	if agent.mode == common.AGENT_MODE_SERVER {
		listener, err := net.Listen("tcp", fmt.Sprintf("%v:%v", agent.conf.Addr, agent.conf.Port))
		if err != nil {
			infof("TcpAgent|Start|Listen|ERROR|err=%v", err)
			return err
		}
		go agent.listener(listener)
	} else if agent.mode == common.AGENT_MODE_CLIENT {
		go agent.connecter()
	}

	infof("tcp agent started")
	return nil
}

func (agent *TcpAgent) resetRunningStat() {
	agent.runningStat = base.RuningStat{
		StartTime: time.Now(),
		StopTime:  agent.runningStat.StopTime,
	}
}

func (agent *TcpAgent) connecter() error {
	ctx := context.Background()
	infof, errorf := agent.getInfof(ctx), agent.getErrorf(ctx)
	for {
		if agent.curConnCount < agent.maxConnCount {
			conn, err := net.Dial("tcp", fmt.Sprintf("%s:%v", agent.conf.Addr, agent.conf.Port))
			if err != nil {
				errorf("TcpAgent|connecter|ERROR|Dial failed, err=%v", err)
			} else {
				connInfo := agent.addConn(conn)
				infof("TcpAgent|connecter|Connection established with server, connid: %v, conninfo: %v", connInfo.id, getConnInfo(conn))
				go agent.receiver(ctx, connInfo)
			}
		}
		// wait 100ms or need conn
		select {
		case <-time.After(time.Millisecond * 100):
		case <-agent.connChan:
		case <-agent.runningCtx.Done():
			return nil
		}
	}
}

func (agent *TcpAgent) listener(listener net.Listener) error {
	ctx := context.Background()
	infof := agent.getInfof(ctx)

	infof("begin listener, addr: %+v", listener.Addr())

	for {
		if agent.curConnCount < agent.maxConnCount {
			conn, err := listener.Accept()
			if err != nil {
				infof("TcpAgent|listener|Accept|ERROR|err=%v", err)
			} else {
				infof("TcpAgent|listener|tcp connection Accept, %v", getConnInfo(conn))

				connInfo := agent.addConn(conn)
				go agent.receiver(ctx, connInfo)
			}
		}
		select {
		case <-time.After(time.Millisecond * 100):
		case <-agent.connChan:
		case <-agent.runningCtx.Done():
			return nil
		}
	}
}

func (agent *TcpAgent) addConn(conn net.Conn) *ConnInfo {
	agent.connMu.Lock()
	defer agent.connMu.Unlock()
	agent.curConnCount += 1

	connInfo := &ConnInfo{
		conn: conn,
		id:   agent.nextConnId,
	}

	agent.conns[connInfo.id] = connInfo
	agent.nextConnId += 1

	if agent.runningStat.FirstConnectionTime.IsZero() {
		agent.runningStat.FirstConnectionTime = time.Now()
	}
	return connInfo
}

func (agent *TcpAgent) delConn(connId int) bool {
	agent.connMu.Lock()
	defer agent.connMu.Unlock()

	if _, ok := agent.conns[connId]; !ok {
		return false
	}
	agent.curConnCount -= 1
	delete(agent.conns, connId)
	return true
}

func (agent *TcpAgent) close1Conn(ctx context.Context, connInfo *ConnInfo) {
	agent.getInfof(ctx)("close conn, connid: %v, conninfo: %v", connInfo.id, getConnInfo(connInfo.conn))

	deleted := agent.delConn(connInfo.id)
	if deleted {
		agent.connChan <- true
	}
	connInfo.conn.Close()
}

func (agent *TcpAgent) closeAllConn(ctx context.Context) {
	agent.getInfof(ctx)("close all conn")

	agent.connMu.Lock()
	defer agent.connMu.Unlock()

	if len(agent.conns) == 0 {
		return
	}

	for connId, connInfo := range agent.conns {
		connInfo.conn.Close()
		delete(agent.conns, connId)
		agent.curConnCount -= 1
	}

	agent.connChan <- true
}

// FIFO cycle find available conn
func (agent *TcpAgent) getAvailableConn() *ConnInfo {
	agent.connMu.Lock()
	defer agent.connMu.Unlock()

	if len(agent.conns) == 0 {
		return nil
	}

	connIds := make([]int, 0, len(agent.conns))
	for id := range agent.conns {
		connIds = append(connIds, id)
	}
	sort.Ints(connIds)

	minId, maxId := connIds[0], connIds[len(connIds)-1]

	if agent.lastSendConnId >= maxId {
		agent.lastSendConnId = minId
		return agent.conns[minId]
	}

	// binary search to find next one
	index := sort.Search(len(connIds), func(index int) bool {
		return connIds[index] > agent.lastSendConnId
	})
	if res, ok := agent.conns[index]; ok {
		agent.lastSendConnId = res.id
		return res
	}
	agent.lastSendConnId = minId
	return agent.conns[minId]
}

func (agent *TcpAgent) receiver(ctx context.Context, connInfo *ConnInfo) {
	infof, errorf := agent.getInfof(ctx), agent.getErrorf(ctx)

	conn := connInfo.conn

	infof("receiver|begin, connid: %v, conninfo: %v", connInfo.id, getConnInfo(conn))

	defer func() {
		agent.close1Conn(ctx, connInfo)
		log.Printf("receiver done, connId: %v, conninfo: %v", connInfo.id, getConnInfo(conn))
	}()

	recvBuffer := make([]byte, TCP_RECV_BUFFER_LEN)
	currentLen := 0
	for {
		n, err := conn.Read(recvBuffer[currentLen:])
		if err != nil {
			infof("Tcp Read failed: %v, currentLen: %v, connId: %v, conn: %v", err, currentLen, connInfo.id, getConnInfo(conn))
			return
		}
		if agent.debug {
			infof("receive len: %v, currentLen : %v, connId: %v, conn: %v", n, currentLen, connInfo.id, getConnInfo(conn))
		}

		agent.runningStat.RecvDataCount += 1
		agent.runningStat.RecvDataTotal += int64(n)

		// decoder failed will close the conn, because the data decode will confuse in the future
		dataList, remainBuffer, err := agent.decoder(recvBuffer[0 : n+currentLen])
		if err != nil {
			errorf("decoder data failed, err: %v, connId: %v, conn: %v", err, connInfo.id, conn)
			return
		}
		for _, data := range dataList {
			agent.recvChan <- &base.RecvNetData{Data: data}
			agent.runningStat.RecvMsgCount += 1
		}
		if len(remainBuffer) > 0 {
			copy(recvBuffer, remainBuffer)
		}
		currentLen = len(remainBuffer)
	}
}

func (agent *TcpAgent) Send(ctx context.Context, data []byte) error {
	errorf := agent.getErrorf(ctx)
	if !agent.Connected() {
		errorf("send failed, agent disconnected")
		return common.GetError(common.ERROR_DISCONNECTED)
	}

	// get connected conn
	connInfo := agent.getAvailableConn()
	if connInfo == nil {
		errorf("get available conninfo failed")
		return common.GetErrorWithMsg(common.ERROR_EMPTY, "no available connection")
	}

	sendData := agent.encoder(data)

	err := agent.send(ctx, connInfo, sendData)
	if err != nil {
		errorf("agent send failed, err: %v, connId: %v, conninfo: %v", err, connInfo.id, getConnInfo(connInfo.conn))
		return err
	}

	return nil
}

func (agent *TcpAgent) send(ctx context.Context, connInfo *ConnInfo, data []byte) error {
	infof, errorf := agent.getInfof(ctx), agent.getErrorf(ctx)

	conn := connInfo.conn

	n, err := conn.Write(data)
	if err != nil {
		errorf("sender|ERR: failed to send data to client, err: %v, connId: %v, conninfo: %v", err, connInfo.id, getConnInfo(conn))
		agent.close1Conn(ctx, connInfo)
		return err
	}
	if agent.debug {
		infof("sender|INFO: connId: %v, connInfo: %v, send len: %v, msg: %v, ", connInfo.id, getConnInfo(conn), n, string(data))
	}
	agent.runningStat.SendMsgCount += 1
	agent.runningStat.SendDataTotal += int64(len(data))

	return nil
}

func (agent *TcpAgent) Receive() chan *base.RecvNetData {
	return agent.recvChan
}

func (agent *TcpAgent) GetRunningStat() base.RuningStat {
	return agent.runningStat
}

func (agent *TcpAgent) Stop() {
	if agent.stop != nil {
		agent.stop()
		agent.stop = nil
		agent.runningStat.StopTime = time.Now()
		agent.closeAllConn(context.Background())
	}
}