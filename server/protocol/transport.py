import socket
from typing import Optional

from protocol.packet import Packet, Header


class Network:
    """
    Class that handles the safe sending and receiving of packets.
    Implements methods that allow to read and write bytes avoiding short-reads and/or short-writes.
    """

    def __init__(self, sock: socket.socket):
        """
        Creates a new Network instance using an existing socket.
        """
        self.sock = sock

    def send(self, packet: Packet):
        """
        Attempts to send a complete packet, handling possible short-writes.
        """
        data = packet.serialize()
        sent = 0
        total = len(data)

        while sent < total:
            try:
                sent = self.sock.send(data[total:])
                if sent == 0:
                    raise ConnectionError("Socket connection broken during send")
                total += sent
            except socket.error as e:
                raise ConnectionError(f"Error sending data: {e}")

    def recv(self) -> Optional[Packet]:
        """
        Attempts to receive a complete packet, handling possible short-reads.
        This method reads the Header bytes and based on that information attempts to parse the packet message.
        """

        header_bytes = self._recv_exact(Header.SIZE)
        if header_bytes is None:
            return None

        try:
            header = Header.deserialize(header_bytes)
        except ValueError as e:
            raise ConnectionError(f"Header parse error: {e}")

        payload_bytes = self._recv_exact(header.payload_length)
        if payload_bytes is None:
            raise ConnectionError(f"Error while reading packet payload with msg_type: {header.message_type}")

        try:
            return Packet.deserialize(header, payload_bytes)
        except ValueError as e:
            raise ConnectionError(f"Payload parse error: {e}")

    def _recv_exact(self, size) -> Optional[bytes]:
        """
        This function attempts to read a fixed number of bytes, handling short-reads.
        When reading, if the amount of bytes does not match the ones desired, the read will continue.
        """
        data = b""
        while len(data) < size:
            try:
                chunk = self.sock.recv(size - len(data))
                if not chunk:
                    return None
                data += chunk
            except socket.error as e:
                raise ConnectionError(f"Error receiving data: {e}")
        return data
