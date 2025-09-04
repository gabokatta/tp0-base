import logging
from typing import Optional

from common.bet_handler import BetHandler
from protocol.packet import (
    Packet, BetStartPacket, BetFinishPacket, BetPacket,
    ErrorPacket, ReplyPacket, GetWinnersPacket
)


class SessionHandler:
    """Handles session state and routing for a single client connection."""

    def __init__(self, bet_handler: BetHandler):
        self.bet_handler = bet_handler
        self.session_id: Optional[int] = None
        self.is_active = False

    def handle_packet(self, packet: Packet) -> Packet:
        """Route packet to appropriate handler based on session state."""
        if isinstance(packet, BetStartPacket):
            return self._handle_session_start(packet)

        if isinstance(packet, GetWinnersPacket):
            return self.bet_handler.handle(packet)

        if not self.is_active:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Session not started")

        if isinstance(packet, (BetPacket, BetFinishPacket)):
            if packet.agency_id != self.session_id:
                return ErrorPacket(ErrorPacket.INVALID_PACKET, "Agency ID mismatch")

            response = self.bet_handler.handle(packet)

            if isinstance(packet, BetFinishPacket) and isinstance(response, ReplyPacket):
                self._end_session()

            return response

        return ErrorPacket(ErrorPacket.INVALID_PACKET, f"Unknown packet type: {type(packet)}")

    def _handle_session_start(self, packet: BetStartPacket) -> Packet:
        """Handle session initialization."""
        if self.is_active:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Session already active")

        self.is_active = True
        self.session_id = packet.agency_id
        logging.info(f"action: session_started | result: success | client_id: {self.session_id}")
        return ReplyPacket(0, "session_active")

    def _end_session(self):
        """Clean up session state."""
        logging.info(f"action: session_ended | result: success | client_id: {self.session_id}")
        self.is_active = False
        self.session_id = None
