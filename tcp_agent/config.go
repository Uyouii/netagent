package tcp_agent

import (
	"crypto/tls"

	"github.com/uyouii/netagent/common"
)

type TcpConfig struct {
	Addr          string
	Port          int
	ConnCount     int // support max conn count
	Mode          common.AgentMode
	Debug         bool
	RecvBufferLen int // if set zero, default is 32k
	UseTls        bool
	TlsConfig     *tls.Config
}

const (
	TCP_DEFAULT_RECV_BUFFER_LEN = 1024 * 32 // 32k, must larger than the max data length
	TCP_MIN_RECV_BUFFER_LEN     = 2048
)

type EncoderFunc func([]byte) []byte
type DecoderFunc func(recvBuffer []byte) (dataList [][]byte, remainBuffer []byte, err error)
