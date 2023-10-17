package tcp_agent

import (
	"encoding/binary"
	"fmt"
	"net"
)

// add 4 byte data len to data begin
func DefaultTcpEncoder(data []byte) []byte {
	const LEN = 4
	sendbuffer := make([]byte, LEN+len(data))
	binary.BigEndian.PutUint32(sendbuffer, uint32(len(data)))
	copy(sendbuffer[LEN:], data)
	return sendbuffer
}

func DefaultTcpDecoder(recvBuffer []byte) ([][]byte, []byte, error) {
	const LEN = 4
	res := [][]byte{}
	for len(recvBuffer) > LEN {
		dataLen := binary.BigEndian.Uint32(recvBuffer[:LEN])
		if len(recvBuffer) < int(dataLen+LEN) {
			break
		}
		recvData := make([]byte, dataLen)
		copy(recvData, recvBuffer[LEN:LEN+int(dataLen)])
		res = append(res, recvData)

		recvBuffer = recvBuffer[LEN+dataLen:]
	}
	return res, recvBuffer, nil
}

func getConnInfo(conn net.Conn) string {
	if conn == nil {
		return ""
	}
	return fmt.Sprintf("conn info: localaddr[%v], remoteaddr[%v]", conn.LocalAddr(), conn.RemoteAddr())
}
