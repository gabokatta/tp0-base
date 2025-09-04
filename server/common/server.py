import logging
import socket

from common.bet_handler import BetHandler
from protocol.packet import BetStartPacket, BetFinishPacket, BetPacket, ErrorPacket, ReplyPacket, Packet
from protocol.transport import Network


class Server:

    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._alive = True
        self._server_socket_closed = False
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._client_socket = None
        self._bet_service = BetHandler()

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communication
        finishes, servers starts to accept new connections again
        """
        while self._alive:
            try:
                self._client_socket = self.__accept_new_connection()
                self.__handle_client_connection()
            except OSError as e:
                if self._alive:
                    logging.error(f"action: accept_connection | result: fail | error: {e}")

        self._cleanup()
        logging.info(f'action: graceful_shutdown | result: success')

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.debug('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.debug(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def __handle_client_connection(self):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        with self._client_socket as s:
            try:
                network = Network(s)
                session_client_id = None
                is_active_session = False

                while True:
                    packet = network.recv()
                    if packet is None:
                        break
                    if isinstance(packet, BetStartPacket):
                        if is_active_session:
                            logging.warning(f"action: receive_start | result: fail | error: session_already_active")
                            network.send(ErrorPacket(ErrorPacket.INVALID_PACKET, "Session already active"))
                        else:
                            is_active_session = True
                            session_client_id = packet.agency_id
                            network.send(ReplyPacket(0, "session_active"))
                    elif isinstance(packet, BetPacket):
                        response = self._process_bet_packet(packet, is_active_session, session_client_id)
                        network.send(response)
                    elif isinstance(packet, BetFinishPacket):
                        response = self._process_finish_packet(packet, is_active_session, session_client_id)
                        network.send(response)
                    else:
                        logging.warning(f"action: receive_message | result: fail | error: unknown_packet_type")
                        response = ErrorPacket(ErrorPacket.INVALID_PACKET, f"Unknown packet type")
                        network.send(response)
                        break
            except ConnectionError as e:
                logging.error(f"action: receive_message | result: fail | error: {e}")
            except Exception as e:
                logging.error(f"action: handle_session | result: fail | error: {e}")

    def _process_bet_packet(self, packet, session_active, session_client_id) -> Packet:
        """Process BetPacket logic."""
        if not session_active:
            logging.warning(f"action: receive_bet | result: fail | error: session_not_started")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Session not started")
        else:
            return self._bet_service.handle_bet_batch(packet, session_client_id)

    def _process_finish_packet(self, packet, session_active, session_client_id):
        """Process BetFinishPacket logic."""
        if not session_active:
            logging.warning(f"action: receive_finish | result: fail | error: session_not_started")
            return ErrorPacket(ErrorPacket.INVALID_PACKET, "Session not started")
        else:
            return self._bet_service.handle_bet_finish(packet, session_client_id)

    def _cleanup(self):

        if self._client_socket:
            try:
                self._client_socket.close()
                logging.info('action: client_socket_shutdown | result: success')
            except OSError as e:
                logging.error(f'action: client_socket_shutdown | result: fail | error: {e}')

        if not self._server_socket_closed:
            try:
                self._server_socket.close()
                self._server_socket_closed = True
                logging.info('action: server_socket_shutdown | result: success')
            except OSError as e:
                logging.error(f'action: server_socket_shutdown | result: fail | error: {e}')

    def shutdown(self):
        logging.debug('action: graceful_shutdown | result: in_progress')
        self._alive = False
        # attempt to close server socket to make server quit waiting new connections.
        if not self._server_socket_closed:
            try:
                self._server_socket.close()
                self._server_socket_closed = True
                logging.info('action: server_socket_forced_close | result: success')
            except OSError as e:
                logging.error(f'action: server_socket_forced_close | result: fail | error: {e}')
