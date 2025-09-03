import logging

from protocol.packet import (Packet, ErrorPacket, BetPacket, ReplyPacket, BetFinishPacket, GetWinnersPacket,
                             ReplyWinnersPacket)
from protocol.data import ProtocolBet
from common.utils import store_bets, Bet


class BetHandler:
    """
    Handles bet packet processing and storage.
    Validates packets, converts to domain objects, and stores using store_bets().
    """

    def handle(self, packet: Packet) -> Packet:

        if not packet:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Failed to handle bet message.")

        if isinstance(packet, BetPacket):
            return self.handle_bets(packet)
        elif isinstance(packet, BetFinishPacket):
            return self.handle_finish(packet)
        elif isinstance(packet, GetWinnersPacket):
            return self.handle_winners(packet)
        else:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Invalid packet type received.")

    @staticmethod
    def handle_bets(packet: BetPacket) -> Packet:
        agency = packet.agency_id
        bets: [Bet] = []
        try:
            parsed = ProtocolBet.to_domain_list(agency, packet.bets)
            bets.extend(parsed)
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | client_id:{agency} | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, "Invalid Bet batch, could not parse.")

        try:
            store_bets(bets)
            logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)} | client_id: {agency}")
            return ReplyPacket(len(bets), "STORED")
        except Exception as e:
            logging.error(
                f"action: apuesta_recibida | result: fail | cantidad: {len(bets)} | client_id_ {agency} | error: {e}"
            )
            return ErrorPacket(ErrorPacket.INVALID_BET, f"Internal server error processing batch of {len(bets)} bets")

    def handle_finish(self, packet: BetFinishPacket) -> Packet:
        raise NotImplementedError("todo")

    def handle_winners(self, packet: GetWinnersPacket) -> Packet:
        raise NotImplementedError("todo")
