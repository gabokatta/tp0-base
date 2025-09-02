import logging

from protocol.packet import Packet, ErrorPacket, BetPacket, ReplyPacket
from protocol.data import ProtocolBet
from common.utils import store_bets, Bet


class BetHandler:

    @staticmethod
    def handle(packet: Packet) -> Packet:

        if not packet:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Failed to handle bet message.")

        if not isinstance(packet, BetPacket):
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Did not receive correct BetPacket.")

        bets: [Bet] = []
        try:
            parsed = ProtocolBet.to_domain_list(packet.agency_id, packet.bets)
            bets.extend(parsed)
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, "Invalid Bet batch, could not parse.")

        try:
            store_bets(bets)
            logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")
            return ReplyPacket(len(bets), "STORED")
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | cantidad: {len(bets)} | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, f"Internal server error processing batch of {len(bets)} bets")
