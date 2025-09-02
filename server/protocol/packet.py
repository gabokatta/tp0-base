from abc import ABC, abstractmethod
from common.utils import Bet
from protocol.data import ProtocolBet
from protocol.utils import ByteReader, ByteWriter

MSG_BET = 0x01
MSG_REPLY = 0x02
MSG_ERROR = 0x03


class Header:
    """
    Packet Header structure (5 Bytes total):
        - 1 Byte: message_type (uint8) - Identifies the packet type
        - 4 Bytes: payload_length (uint32) - Length of the following payload

    The header precedes every packet and indicates what type of message follows
    and how many bytes the payload contains.
    """
    SIZE = 5

    def __init__(self, message_type: int, payload_length: int):
        self.message_type = message_type
        self.payload_length = payload_length

    def serialize(self) -> bytes:
        writer = ByteWriter()
        writer.write_uint8(self.message_type)
        writer.write_uint32(self.payload_length)
        return writer.get_bytes()

    @classmethod
    def deserialize(cls, data: bytes) -> 'Header':
        if len(data) != Header.SIZE:
            raise ValueError("Invalid header size")

        reader = ByteReader(data)
        message_type = reader.read_uint8()
        payload_length = reader.read_uint32()

        return cls(message_type, payload_length)


class Packet(ABC):
    """Base Abstract Class for Packet Types."""

    @abstractmethod
    def get_message_type(self) -> int:
        """Gets the message type."""
        pass

    @abstractmethod
    def serialize_payload(self) -> bytes:
        """Serializes payload into Bytes."""
        pass

    @classmethod
    @abstractmethod
    def deserialize_payload(cls, data: bytes) -> 'Packet':
        """Parses the bytes into a concrete Packet object."""
        pass

    def serialize(self) -> bytes:
        """Complete packet serialization including header and payload."""
        payload = self.serialize_payload()
        header = Header(self.get_message_type(), len(payload))
        return header.serialize() + payload

    @classmethod
    def deserialize(cls, header: Header, payload: bytes) -> 'Packet':
        """Based on the header, deserializes it into a concrete Packet object."""

        packet_classes = {
            MSG_BET: BetPacket,
            MSG_REPLY: ReplyPacket,
            MSG_ERROR: ErrorPacket,
        }

        packet_class = packet_classes.get(header.message_type, None)
        if not packet_class:
            raise ValueError(f"Unknown message type: {header.message_type}")

        return packet_class.deserialize_payload(payload)


class BetPacket(Packet):
    """
    Bet Packet Payload structure:
        - 1 Byte: agency_id (uint8) - Identifier for the betting agency
        - Bet object: See ProtocolBet class for detailed structure

    This packet contains betting information submitted by agencies.
    The bet data follows the ProtocolBet format.
    """

    def __init__(self, agency_id: int, bet: ProtocolBet):
        self.agency_id = agency_id
        self.bet = bet

    def get_message_type(self) -> int:
        return MSG_BET

    def serialize_payload(self) -> bytes:
        writer = ByteWriter()
        writer.write_uint8(self.agency_id)
        writer.write_bytes(self.bet.to_bytes())
        return writer.get_bytes()

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'BetPacket':
        reader = ByteReader(data)
        agency_id = reader.read_uint8()
        bet, _ = ProtocolBet.from_bytes(reader.data, reader.offset)
        return cls(agency_id, bet)

    @classmethod
    def from_domain_bet(cls, bet: Bet) -> 'BetPacket':
        """Creates a BetPacket object from a bet object."""
        protocol_bet = ProtocolBet.from_domain(bet)
        return BetPacket(int(bet.agency), protocol_bet)


class ReplyPacket(Packet):
    """
    Reply Packet Payload structure:
        - 4 Bytes: done_count (uint32) - Number of processed operations
        - Variable: message - Length-prefixed string (1 byte length + UTF-8 bytes)

    This packet is sent as a response to operations, indicating success/failure
    and providing additional information.
    """

    def __init__(self, done_count: int, msg: str = ""):
        self.done_count = done_count
        self.msg = msg

    def get_message_type(self) -> int:
        return MSG_REPLY

    def serialize_payload(self) -> bytes:
        writer = ByteWriter()
        writer.write_uint32(self.done_count)
        writer.write_string(self.msg)
        return writer.get_bytes()

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'ReplyPacket':
        reader = ByteReader(data)
        done_count = reader.read_uint32()
        message = reader.read_string()
        return cls(done_count, message)


class ErrorPacket(Packet):
    """
    Error Packet Payload structure:
        - 1 Byte: error_code (uint8) - Specific error identifier
        - Variable: message - Length-prefixed string (1 byte length + UTF-8 bytes)

    This packet is used to communicate error conditions with specific error codes
    and descriptive messages.
    """

    INVALID_PACKET = 0x01
    INVALID_BET = 0x02

    CODES = {
        INVALID_PACKET: "BAD_PACKET",
        INVALID_BET: "BAD_BET",
    }

    def __init__(self, error_code: int, message: str = ""):
        self.error_code = error_code
        self.message = message

    def get_message_type(self) -> int:
        return MSG_ERROR

    def serialize_payload(self) -> bytes:
        writer = ByteWriter()
        writer.write_uint8(self.error_code)
        writer.write_string(self.message)
        return writer.get_bytes()

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'ErrorPacket':
        reader = ByteReader(data)
        error_code = reader.read_uint8()
        message = reader.read_string()
        return cls(error_code, message)
