import logging
import threading
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

    def __init__(self, agency_amount, thread_shutdown: threading.Event):
        self.agency_amount: int = agency_amount
        self.lottery_is_done: bool = False
        self.winners: Dict[int, List[str]] = {}
        self.ready_agencies: Set[int] = set()
        self.thread_shutdown = thread_shutdown

        # para sincronizar
        self.lottery_var = threading.Condition()
        self._file_lock = threading.Lock()

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

    def _handle_bets(self, packet: BetPacket) -> Packet:
        """Handle incoming bet packets - store bets and return confirmation."""
        agency = packet.agency_id
        logging.info(f"action: handle_bets | result: in_progress | client_id: {agency}" +
                     f" | bet_count: {len(packet.bets)}")

        try:
            parsed_bets = ProtocolBet.to_domain_list(agency, packet.bets)
        except Exception as e:
            logging.error(f"action: apuesta_recibida | result: fail | client_id: {agency} |" +
                          f" error: bet_parsing | details: {e}")
            return ErrorPacket(ErrorPacket.INVALID_BET, "Invalid Bet batch, could not parse.")

        try:
            #  Accedemos a sección crítica de CSV usando un wrapper que pide un lock y lo suelta al terminar de operar.
            self._locked_store_bets(parsed_bets)
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
        logging.info(f"action: finish_ack | result: in_progress | client_id: {agency}")
        try:
            with self.lottery_var:
                self.ready_agencies.add(agency)  # estado compartido modificado usando el RLOCK de la condVar.
                if self._should_start_lottery():
                    self._start_lottery()   # Déja cargado los ganadores y queda como read-only. (atómico)
                    self.lottery_var.notify_all()   # despierta a todos los que están esperando ganadores.
                logging.info(f"action: finish_ack | result: success | client_id: {agency} |" +
                             f" ready_count: {len(self.ready_agencies)}/{self.agency_amount}")
            return ReplyPacket(len(self.ready_agencies), f"Agency {agency} registered as ready")
        except Exception as e:
            logging.error(f"action: finish_ack | result: fail | client_id: {agency} | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Internal server error processing finish notification")

    def _handle_winners(self, packet: GetWinnersPacket) -> Packet:
        """Handle winners query from agencies."""
        agency = packet.agency_id
        try:
            with self.lottery_var:
                while not self.lottery_is_done:     # estado compartido leído usando el RLOCK de la condVar.
                    self.lottery_var.wait()
                    # si fuimos despertados pero el shutdown esta encendido, nos vamos.
                    if self.thread_shutdown.is_set():
                        logging.info(f"action: winner_request | result: fail | client_id: {agency}")
                        return ErrorPacket(ErrorPacket.INVALID_PACKET, "Server shutting down")
            #   esta operación se vuelve read-only de un mapa inmutable.
            agency_winners = self.winners.get(agency, [])
            logging.info(f"action: winner_request | result: success | client_id: {agency} |" +
                         f" winner_count: {len(agency_winners)}")
            return ReplyWinnersPacket(agency, [int(doc) for doc in agency_winners])
        except ValueError as e:
            logging.error(f"action: winner_request | result: fail | client_id: {agency} |" +
                          f" error: document_conversion | details: {e}")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Error converting winner documents")
        except Exception as e:
            logging.error(f"action: winner_request | result: fail | client_id: {agency} | error: {e}")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Internal server error processing winners query")

    def _should_start_lottery(self) -> bool:
        """Check if all agencies finished and lottery hasn't started yet."""
        all_agencies_ready = len(self.ready_agencies) == self.agency_amount
        return all_agencies_ready and not self.lottery_is_done

    def _start_lottery(self) -> None:
        """
        Perform lottery calculation. Called only when all agencies are ready.
        After completion, self.winners becomes immutable and self.lottery_is_done = True.
        """
        logging.info("action: sorteo | result: in_progress")
        try:
            self.winners = {agency_id: [] for agency_id in range(1, self.agency_amount + 1)}
            bets = list(self._locked_load_bets())

            for bet in bets:
                if has_won(bet):
                    agency_id = bet.agency
                    self.winners[agency_id].append(bet.document)

            self.lottery_is_done = True
            logging.info("action: sorteo | result: success")
        except Exception as e:
            logging.error(f"action: sorteo | result: fail | error: {e}")
            self.winners = {}
            raise

    def _locked_store_bets(self, bets):
        """Thread-safe wrapper for store_bets using file lock."""
        with self._file_lock:
            store_bets(bets)

    def _locked_load_bets(self):
        """Thread-safe wrapper for load_bets using file lock."""
        with self._file_lock:
            return list(load_bets())
