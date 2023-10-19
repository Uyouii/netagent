package base

import (
	"context"
	"time"
)

type RuningStat struct {
	StartTime           time.Time
	StopTime            time.Time
	FirstConnectionTime time.Time
	SendMsgCount        int64
	SendDataTotal       int64
	RecvMsgCount        int64
	RecvDataCount       int64
	RecvDataTotal       int64
}

type RecvNetData struct {
	Data     []byte
	ConnInfo interface{}
}

type NetAgent interface {
	Start() error
	Stop()
	Send(ctx context.Context, data []byte) error
	SendByConn(ctx context.Context, connInfo interface{}, data []byte) error
	Receive() chan *RecvNetData
	Connected() bool
	ConnectedCnt() int
	GetRunningStat() RuningStat
}
