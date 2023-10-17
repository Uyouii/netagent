package tcp_agent

import "github.com/uyouii/netagent/common"

type TcpConfig struct {
	Addr      string
	Port      int
	ConnCount int // support max conn count
	Mode      common.AgentMode
	Debug     bool
}

const TCP_RECV_BUFFER_LEN = 1024 * 32 // 32k, must larger than the max data length

type EncoderFunc func([]byte) []byte
type DecoderFunc func(recvBuffer []byte) (dataList [][]byte, remainBuffer []byte, err error)
