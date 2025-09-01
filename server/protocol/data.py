from protocol.utils import ByteReader, ByteWriter
from common.utils import Bet


class ProtocolBet:
    """
    Network Layer representation of the Bet object.

    Structure:
        - 1 Byte: name_len + actual name bytes (UTF-8)
        - 1 Byte: lastname_len + actual lastname bytes (UTF-8)
        - 4 Bytes: document (uint32)
        - 4 Bytes: birthdate (uint32, YYYYMMDD format)
        - 2 Bytes: number (uint16)

    This class handles the binary serialization/deserialization of bet data
    for network transmission, converting between domain objects and byte format.
    """

    def __init__(self, first_name: str, last_name: str, document: int, birthdate: int, number: int):
        self.first_name = first_name
        self.last_name = last_name
        self.document = document
        self.birthdate = birthdate  # YYYYMMDD
        self.number = number

    @classmethod
    def from_domain(cls, bet: Bet) -> 'ProtocolBet':
        """Creates a ProtocolBet object from a Bet object."""

        # this is done to go from YYYY-MM-DD to YYYYMMDD as INT representation.
        birthdate_as_int = bet.birthdate.year * 10000 + bet.birthdate.month * 100 + bet.birthdate.day

        if not bet.document.isnumeric():
            raise ValueError(f'Document must be an integer: {bet.document}')

        return ProtocolBet(
            bet.first_name,
            bet.last_name,
            int(bet.document),
            birthdate_as_int,
            bet.number
        )

    def to_domain(self, agency_id: int) -> Bet:
        """Creates a domain object from a Bet object."""

        # this is done to go from the YYYYMMDD format to YYYY-MM-DD
        year = self.birthdate // 10000
        month = (self.birthdate // 100) % 100
        day = self.birthdate % 100
        birthdate = f"{year:04d}-{month:02d}-{day:02d}"

        return Bet(
            str(agency_id),
            self.first_name,
            self.last_name,
            str(self.document),
            birthdate,
            str(self.number)
        )

    def to_bytes(self) -> bytes:
        """Transforms a network Bet into the byte representation using ByteWriter."""
        writer = ByteWriter()
        writer.write_string(self.first_name)
        writer.write_string(self.last_name)
        writer.write_uint32(self.document)
        writer.write_uint32(self.birthdate)
        writer.write_uint16(self.number)
        return writer.get_bytes()

    @classmethod
    def from_bytes(cls, data: bytes, offset: int = 0) -> tuple['ProtocolBet', int]:
        """Reads a bet from bytes and returns the current offset using ByteReader."""
        reader = ByteReader(data, offset)

        first_name = reader.read_string()
        last_name = reader.read_string()
        document = reader.read_uint32()
        birthdate = reader.read_uint32()
        number = reader.read_uint16()

        return cls(first_name, last_name, document, birthdate, number), reader.offset
