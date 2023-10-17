package netagent

import (
	"github.com/uyouii/netagent/base"
	"github.com/uyouii/netagent/common"
	"github.com/uyouii/netagent/tcp_agent"
)

func NewTcpAgent(tcpConfig *tcp_agent.TcpConfig,
	encoder tcp_agent.EncoderFunc, decoder tcp_agent.DecoderFunc,
	getInfof common.GetLogfFunc, getErrorf common.GetLogfFunc) (base.NetAgent, error) {
	return tcp_agent.NewTcpAgent(tcpConfig, encoder, decoder, getInfof, getErrorf)
}
