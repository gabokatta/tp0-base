import logging
import socket

from common.bet_handler import BetHandler
from protocol.packet import ErrorPacket
from protocol.transport import Network
from common.session import SessionHandler


class Server:

    def __init__(self, port, listen_backlog, agency_amount):
        # Initialize server socket
        self._alive = True
        self._server_socket_closed = False
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._client_socket = None
        self._bet_service = BetHandler(agency_amount)

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
                self._handle_client_connection()
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

    def _handle_client_connection(self):
        """Handle complete client session."""
        with self._client_socket as s:
            try:
                network = Network(s)
                session = SessionHandler(self._bet_service)

                while self._alive:
                    packet = network.recv()
                    if packet is None:
                        break

                    logging.debug(f"action: receive_message | result: in_progress | ip: {s.getpeername()}")

                    response = session.handle_packet(packet)
                    network.send(response)

                    if isinstance(response, ErrorPacket) and response.error_code == ErrorPacket.INVALID_PACKET:
                        break

            except ConnectionError as e:
                logging.error(f"action: receive_message | result: fail | error: {e}")
            except Exception as e:
                logging.error(f"action: handle_session | result: fail | error: {e}")

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
