import logging
import socket
import threading

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

        self._active_threads = []
        self._threads_lock = threading.Lock()
        self._thread_shutdown = threading.Event()

        self._bet_service = BetHandler(agency_amount, self._thread_shutdown)

    def run(self):
        """
        Concurrent Server loop

        Server that accepts new connections and creates a thread
        for each client connection
        """
        while self._alive:
            try:
                client_socket = self.__accept_new_connection()
                if client_socket:

                    client_thread = threading.Thread(
                        target=self._handle_client_connection,
                        args=(client_socket,)
                    )
                    client_thread.daemon = False  # para manejar manual el apagado.
                    client_thread.start()

                    with self._threads_lock:
                        self._active_threads.append(client_thread)
                        self._active_threads = [t for t in self._active_threads if t.is_alive()]

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
        try:
            # Connection arrived
            logging.debug('action: accept_connections | result: in_progress')
            c, addr = self._server_socket.accept()
            logging.debug(f'action: accept_connections | result: success | ip: {addr[0]}')
            return c
        except OSError:
            return None

    def _handle_client_connection(self, client_socket):
        """Handle complete client session in separate thread."""
        with client_socket as s:
            try:
                s.settimeout(1.0)
                network = Network(s)
                session = SessionHandler(self._bet_service)

                while not self._thread_shutdown.is_set():
                    try:
                        packet = network.recv()
                        if packet is None:
                            break

                        logging.debug(f"action: receive_message | result: in_progress | ip: {s.getpeername()}")

                        response = session.handle_packet(packet)
                        network.send(response)

                        if isinstance(response, ErrorPacket) and response.error_code == ErrorPacket.INVALID_PACKET:
                            break
                    except socket.timeout:
                        continue
                    except ConnectionError:
                        break
                    except OSError:
                        break
            except ConnectionError as e:
                logging.error(f"action: receive_message | result: fail | error: {e}")
            except Exception as e:
                logging.error(f"action: handle_session | result: fail | error: {e}")

    def _cleanup(self):
        with self._threads_lock:
            active_threads = self._active_threads

        for thread in active_threads:
            if thread.is_alive():
                thread.join()

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
        logging.debug('action: wake_up_threads | result: in_progress')
        self._thread_shutdown.set()
        logging.debug('action: wake_up_winner_gets | result: in_progress')
        with self._bet_service.lottery_var:
            self._bet_service.lottery_var.notify_all()
        # attempt to close server socket to make server quit waiting new connections.
        if not self._server_socket_closed:
            try:
                self._server_socket.close()
                self._server_socket_closed = True
                logging.info('action: server_socket_forced_close | result: success')
            except OSError as e:
                logging.error(f'action: server_socket_forced_close | result: fail | error: {e}')
