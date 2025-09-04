package protocol

import (
	"bytes"
	"fmt"
	"github.com/7574-sistemas-distribuidos/docker-compose-init/client/shutdown"
	"io"
	"net"
	"time"
)

// Network handles TCP network communication with the server,
// including connection management and packet transmission.
type Network struct {
	serverAddress string
	conn          net.Conn
	signal        *shutdown.SignalHandler
}

// NewNetwork creates a new Network instance with the specified server address.
func NewNetwork(serverAddress string, signal *shutdown.SignalHandler) *Network {
	return &Network{
		serverAddress: serverAddress,
		signal:        signal,
	}
}

// Connect establishes a TCP connection to the server.
// Returns an error if the connection cannot be established.
func (n *Network) connect() error {
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

// Disconnect closes the current connection to the server.
// Returns an error if the connection cannot be closed properly.
func (n *Network) Disconnect() error {
	if n.conn != nil {
		err := n.conn.Close()
		n.conn = nil
		return err
	}
	return nil
}

// SendStartBet sends the StartBetPacket to initiate a betting session.
// Establishes connection and sends the start packet.
func (n *Network) SendStartBet(clientID string) (Packet, error) {
	packet, err := NewBetStartPacket(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to create StartBet packet: %w", err)
	}

	if err := n.connect(); err != nil {
		return nil, err
	}

	if err := n.send(packet); err != nil {
		return nil, err
	}

	response, err := n.recv()
	if err != nil {
		return nil, err
	}

	return response, nil
}

// SendBetBatch sends a batch of bets to the server and returns the response packet.
// Creates the BetPacket internally. Assumes connection is already established.
func (n *Network) SendBetBatch(clientID string, bets []Bet) (Packet, error) {
	if n.conn == nil {
		return nil, fmt.Errorf("connection not established, call SendStartBet() first")
	}

	packet, err := NewBetPacket(clientID, bets)
	if err != nil {
		return nil, fmt.Errorf("failed to create bet packet: %w", err)
	}

	if err := n.send(packet); err != nil {
		return nil, err
	}

	response, err := n.recv()
	if err != nil {
		return nil, err
	}

	return response, nil
}

// SendFinishBet sends the FinishBet notification to the server.
// Creates the BetFinishPacket internally. Assumes connection is already established.
// This closes the betting session
func (n *Network) SendFinishBet(clientID string) (Packet, error) {
	defer func() { _ = n.Disconnect() }()
	if n.conn == nil {
		return nil, fmt.Errorf("connection not established, call SendStartBet() first")
	}

	packet, err := NewBetFinishPacket(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to create betFinish packet: %w", err)
	}

	if err := n.send(packet); err != nil {
		return nil, err
	}

	response, err := n.recv()
	if err != nil {
		return nil, err
	}

	return response, nil
}

// SendWinnersRequest sends the GetWinnersPacket to the server.
// Creates the GetWinnersPacket internally and handles connection lifecycle.
// The timeout parameters helps avoid waiting forever for the server.
func (n *Network) SendWinnersRequest(clientID string, timeout time.Duration) (Packet, error) {
	defer func() { _ = n.Disconnect() }()

	packet, err := NewGetWinnersPacket(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to create GetWinnersPacket packet: %w", err)
	}

	if err := n.connect(); err != nil {
		return nil, err
	}

	deadline := time.Now().Add(timeout)
	if err := n.conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set connection deadline: %w", err)
	}

	if err := n.send(packet); err != nil {
		return nil, err
	}

	response, err := n.recv()
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Send writes a packet to the network connection.
// Returns an error if the connection is closed or packet encoding fails.
func (n *Network) send(packet Packet) error {
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

// Recv reads a packet from the network connection.
// Returns an error if the connection is closed or packet decoding fails.
func (n *Network) recv() (Packet, error) {
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

// recvExact reads exactly nBytes from the network connection.
// The main objective is to avoid shortReads.
// Returns an error if the connection is closed before reading all bytes.
func (n *Network) recvExact(nBytes int) ([]byte, error) {
	buf := make([]byte, nBytes)
	bytesRead := 0

	for bytesRead < nBytes {

		if n.signal.ShouldShutdown() {
			return nil, fmt.Errorf("operation cancelled due to shutdown signal")
		}

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

// writeExact writes all bytes from data to the network connection.
// The main objective is to avoid shortWrites.
// Returns the number of bytes written and any error encountered.
func (n *Network) writeExact(data []byte) (int, error) {
	bytesWritten := 0
	totalBytes := len(data)

	for bytesWritten < totalBytes {

		if n.signal.ShouldShutdown() {
			return bytesWritten, fmt.Errorf("operation cancelled due to shutdown signal")
		}

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
