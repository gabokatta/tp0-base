import logging

from protocol.packet import Packet, ErrorPacket, BetPacket, ReplyPacket
from common.utils import store_bets


class BetHandler:

    @staticmethod
    def handle(packet: Packet) -> Packet:

        if not packet:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Failed to handle bet message.")

        if not isinstance(packet, BetPacket):
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Did not receive correct BetPacket.")

        try:
            bet = packet.bet.to_domain(packet.agency_id)
            store_bets([bet])
            logging.info(f"action: apuesta_almacenada | result: success | dni: {bet.document} | numero: {bet.number}")
            return ReplyPacket(1, "STORED")
        except ValueError as e:
            logging.error(f"action: process_bet | result: fail | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, str(e))
        except Exception as e:
            logging.exception(f"action: process_bet | result: fail | unexpected: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, "Internal server error processing bet")
