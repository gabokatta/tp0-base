import logging
from typing import Dict, List, Set

from protocol.packet import (Packet, ErrorPacket, BetPacket, ReplyPacket, BetFinishPacket, GetWinnersPacket,
                             ReplyWinnersPacket)
from protocol.data import ProtocolBet
from common.utils import store_bets, load_bets, has_won


class BetHandler:
    """
    Handles bet packet processing and storage.
    Validates packets, converts to domain objects, and stores using store_bets().
    """

    def __init__(self, agency_amount: int):
        self.ready_agencies: Set[int] = set()
        self.agency_amount = agency_amount
        self.winners: Dict[int, List[str]] = {}
        self.lottery_completed = False

    def handle(self, packet: Packet) -> Packet:
        if not packet:
            logging.error("action: handle_packet | result: fail | error: null_packet")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Failed to handle bet message.")

        if isinstance(packet, BetPacket):
            return self._handle_bets(packet)
        elif isinstance(packet, BetFinishPacket):
            return self._handle_finish(packet)
        elif isinstance(packet, GetWinnersPacket):
            return self._handle_winners(packet)
        else:
            logging.error(f"action: handle_packet | result: fail | packet_type: {type(packet)} | error: unknown_type")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Invalid packet type received.")

    @staticmethod
    def _handle_bets(packet: BetPacket) -> Packet:
        """Handle incoming bet packets - store bets and return confirmation."""
        agency = packet.agency_id
        logging.debug(f"action: handle_bets | result: in_progress | client_id: {agency}" +
                      f" | bet_count: {len(packet.bets)}")

        try:
            parsed_bets = ProtocolBet.to_domain_list(agency, packet.bets)
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | client_id: {agency} |" +
                          f" error: bet_parsing | details: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, "Invalid Bet batch, could not parse.")

        try:
            store_bets(parsed_bets)
            logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(parsed_bets)} |" +
                         f" client_id: {agency}")
            return ReplyPacket(len(parsed_bets), "STORED")
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | cantidad: {len(parsed_bets)} |" +
                          f" client_id: {agency} | error: storage | details: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, f"Internal error processing batch of {len(parsed_bets)} bets")

    def _handle_finish(self, packet: BetFinishPacket) -> Packet:
        """Handle finish notification from agencies."""
        agency = packet.agency_id
        logging.debug(f"action: handle_finish | result: in_progress | client_id: {agency}")
        try:
            self.ready_agencies.add(agency)
            logging.info(f"action: finish_ack | result: success | client_id: {agency} |" +
                         f" ready_count: {len(self.ready_agencies)}/{self.agency_amount}")

            if self._should_start_lottery():
                self._start_lottery()

            return ReplyPacket(len(self.ready_agencies), f"Agency {agency} registered as ready")
        except Exception as e:
            logging.error(f"action: finish_ack | result: fail | client_id: {agency} | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Internal server error processing finish notification")

    def _handle_winners(self, packet: GetWinnersPacket) -> Packet:
        """Handle winners query from agencies."""
        agency = packet.agency_id
        logging.debug(f"action: handle_winners | result: in_progress | client_id: {agency}")

        try:
            if not self.lottery_completed:
                ready_count = len(self.ready_agencies)
                logging.info(f"action: consulta_ganadores | result: lottery_not_ready | client_id: {agency} |" +
                             f" ready_agencies: {ready_count}/{self.agency_amount}")
                return ErrorPacket(ErrorPacket.LOTTERY_NOT_DONE, f"Lottery not completed.")

            agency_winners = self.winners.get(agency, [])
            logging.info(f"action: consulta_ganadores | result: success | client_id: {agency} |" +
                         f" winner_count: {len(agency_winners)}")

            winner_documents = [int(doc) for doc in agency_winners]
            return ReplyWinnersPacket(agency, winner_documents)

        except ValueError as e:
            logging.error(f"action: consulta_ganadores | result: fail | client_id: {agency} |" +
                          f" error: document_conversion | details: {e}")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Error converting winner documents")
        except Exception as e:
            logging.error(f"action: consulta_ganadores | result: fail | client_id: {agency} | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Internal server error processing winners query")

    def _should_start_lottery(self) -> bool:
        """Check if conditions are met to start the lottery."""
        return len(self.ready_agencies) == self.agency_amount and not self.lottery_completed

    def _start_lottery(self) -> None:
        """Perform the lottery calculation and cache results."""
        logging.debug("action: sorteo | result: in_progress")

        try:
            self.winners = {agency_id: [] for agency_id in range(1, self.agency_amount + 1)}
            bets = list(load_bets())
            total_bets = len(bets)

            logging.debug(f"action: sorteo | result: processing | total_bets: {total_bets}")

            for bet in bets:
                if has_won(bet):
                    agency_id = bet.agency
                    self.winners[agency_id].append(bet.document)
                    logging.debug(f"action: sorteo | result: winner_found | agency_id: {agency_id}" +
                                  f" | document: {bet.document}")

            self.lottery_completed = True
        except Exception as e:
            logging.error(f"action: sorteo | result: fail | error: {e}")
            self.winners = {}
            self.lottery_completed = False
            raise
