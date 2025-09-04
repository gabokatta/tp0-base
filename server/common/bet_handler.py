import logging

from protocol.packet import Packet, ErrorPacket, BetPacket, ReplyPacket, BetFinishPacket
from protocol.data import ProtocolBet
from common.utils import store_bets, Bet


class BetHandler:
    """
    Handles bet packet processing and storage.
    Validates packets, converts to domain objects, and stores using store_bets().
    """

    def handle_bet_batch(self, packet: BetPacket, session_client_id) -> Packet:
        if packet.agency_id != session_client_id:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "tried to process batch from invalid client.")
        else:
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

    def handle_bet_finish(self, packet: BetFinishPacket, session_client_id) -> Packet:
        if packet.agency_id != session_client_id:
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "asked to finish from invalid client.")
        else:
            return ReplyPacket(0, "SESSION_FINISHED")
