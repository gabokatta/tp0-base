package protocol

import (
	"bytes"
	"io"
	"net"
)

type Network struct {
	serverAddress string
	conn          net.Conn
}

func NewNetwork(serverAddress string) *Network {
	return &Network{serverAddress: serverAddress}
}

func (n *Network) Connect() error {
	if n.conn != nil {
		return nil
	}

	conn, err := net.Dial("tcp", n.serverAddress)
	if err != nil {
		return err
	}

	n.conn = conn
	return nil
}

func (n *Network) Disconnect() error {
	if n.conn != nil {
		err := n.conn.Close()
		n.conn = nil
		return err
	}
	return nil
}

func (n *Network) IsConnected() bool {
	return n.conn != nil
}

func (n *Network) SendBet(clientID string, bet Bet) (Packet, error) {
	packet, err := NewBetPacket(clientID, bet)
	if err != nil {
		return nil, err
	}

	if err := n.Connect(); err != nil {
		return nil, err
	}

	// lo ignoro ya que estamos en pleno shutdown.
	defer func() { _ = n.Disconnect() }()

	if err := n.Send(packet); err != nil {
		return nil, err
	}

	response, err := n.Recv()
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (n *Network) Send(packet Packet) error {
	if n.conn == nil {
		return net.ErrClosed
	}

	var buf bytes.Buffer
	if err := WritePacket(&buf, packet); err != nil {
		return err
	}
	_, err := n.writeExact(buf.Bytes())
	return err
}

func (n *Network) Recv() (Packet, error) {
	if n.conn == nil {
		return nil, net.ErrClosed
	}

	headerBytes, err := n.recvExact(HeaderSize)
	if err != nil {
		return nil, err
	}

	header, err := DeserializeHeader(headerBytes)
	if err != nil {
		return nil, err
	}

	payloadBytes, err := n.recvExact(int(header.PayloadLength))
	if err != nil {
		return nil, err
	}

	return DeserializePayload(header.MessageType, payloadBytes)
}

func (n *Network) recvExact(nBytes int) ([]byte, error) {
	buf := make([]byte, nBytes)
	bytesRead := 0

	for bytesRead < nBytes {
		n, err := n.conn.Read(buf[bytesRead:])
		if err != nil {
			if err == io.EOF && bytesRead > 0 {
				return nil, io.ErrUnexpectedEOF
			}
			return nil, err
		}
		if n == 0 {
			return nil, io.EOF
		}
		bytesRead += n
	}

	return buf, nil
}

func (n *Network) writeExact(data []byte) (int, error) {
	bytesWritten := 0
	totalBytes := len(data)

	for bytesWritten < totalBytes {
		n, err := n.conn.Write(data[bytesWritten:])
		if err != nil {
			return bytesWritten, err
		}
		if n == 0 {
			return bytesWritten, io.ErrShortWrite
		}
		bytesWritten += n
	}

	return bytesWritten, nil
}
