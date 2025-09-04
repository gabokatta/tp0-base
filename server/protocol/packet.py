from abc import ABC, abstractmethod
from common.utils import Bet
from protocol.data import ProtocolBet
from protocol.utils import ByteReader, ByteWriter

MSG_BET_START = 0X01
MSG_BET = 0x02
MSG_BET_FINISH = 0X03
MSG_REPLY = 0x04
MSG_ERROR = 0x05


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
            MSG_BET_START: BetStartPacket,
            MSG_BET: BetPacket,
            MSG_BET_FINISH: BetFinishPacket,
            MSG_REPLY: ReplyPacket,
            MSG_ERROR: ErrorPacket,
        }

        packet_class = packet_classes.get(header.message_type, None)
        if not packet_class:
            raise ValueError(f"Unknown message type: {header.message_type}")

        return packet_class.deserialize_payload(payload)


class BetStartPacket(Packet):
    """
    Bet Start Packet Payload structure:
        - 1 Byte: agency_id (uint8) - Identifier for the betting agency

    This packet signals to the server that it should prepare to receive multiple
    BetPacket messages.
    """

    def __init__(self, agency_id: int):
        self.agency_id: int = agency_id

    def get_message_type(self) -> int:
        return MSG_BET_START

    def serialize_payload(self) -> bytes:
        raise NotImplementedError("Server has no need to serialize BetStartPacket.")

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'BetStartPacket':
        reader = ByteReader(data)
        agency_id = reader.read_uint8()
        return cls(agency_id)


class BetFinishPacket(Packet):
    """
    BetFinish Packet Payload structure:
        - 1 Byte: agency_id (uint8) - Identifier for the betting agency
    This packet signals that an agency has finished sending bets.
    """

    def __init__(self, agency_id: int):
        self.agency_id: int = agency_id

    def get_message_type(self) -> int:
        return MSG_BET_FINISH

    def serialize_payload(self) -> bytes:
        raise NotImplementedError("Server has no need to serialize BetFinishPacket.")

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'BetFinishPacket':
        reader = ByteReader(data)
        agency_id = reader.read_uint8()
        return cls(agency_id)


class BetPacket(Packet):
    """
    Bet Packet Payload structure:
        - 1 Byte: agency_id (uint8) - Identifier for the betting agency
        - 4 Byte: bet_n (uint32) - Amount of bets
        - ProtocolBet object list: See ProtocolBet class for detailed structure

    This packet contains betting information submitted by agencies.
    The bet data follows the ProtocolBet format.
    """

    def __init__(self, agency_id: int, bets: [ProtocolBet]):
        self.agency_id: int = agency_id
        self.bets: [ProtocolBet] = bets

    def get_message_type(self) -> int:
        return MSG_BET

    def serialize_payload(self) -> bytes:
        raise NotImplementedError("Server has no need to serialize BetPacket.")

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'BetPacket':
        bets = []
        reader = ByteReader(data)
        agency_id = reader.read_uint8()
        bet_amount = reader.read_uint32()
        for _ in range(bet_amount):
            bet, new_offset = ProtocolBet.from_bytes(reader.data, reader.offset)
            reader.offset = new_offset
            bets.append(bet)
        return cls(agency_id, bets)


class ReplyPacket(Packet):
    """
    Reply Packet Payload structure:
        - 4 Bytes: done_count (uint32) - Number of processed operations
        - Variable: message - Length-prefixed string (1 byte length + UTF-8 bytes)

    This packet is sent as a response to operations, indicating success/failure
    and providing additional information.
    """

    def __init__(self, done_count: int, msg: str = ""):
        self.done_count: int = done_count
        self.msg: str = msg

    def get_message_type(self) -> int:
        return MSG_REPLY

    def serialize_payload(self) -> bytes:
        writer = ByteWriter()
        writer.write_uint32(self.done_count)
        writer.write_string(self.msg)
        return writer.get_bytes()

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'ReplyPacket':
        raise NotImplementedError("Client has no need to send ReplyPacket.")


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
        self.error_code: int = error_code
        self.message: str = message

    def get_message_type(self) -> int:
        return MSG_ERROR

    def serialize_payload(self) -> bytes:
        writer = ByteWriter()
        writer.write_uint8(self.error_code)
        writer.write_string(self.message)
        return writer.get_bytes()

    @classmethod
    def deserialize_payload(cls, data: bytes) -> 'ErrorPacket':
        raise NotImplementedError("Client has no need to send ErrorPacket.")
