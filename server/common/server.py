import socket
import logging

CONNECTION_WAIT_TIME = 1.0


class Server:

    def __init__(self, port, listen_backlog):
        # Initialize server socket
        self._alive = True
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.settimeout(CONNECTION_WAIT_TIME)
        self._server_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)

    def run(self):
        """
        Dummy Server loop

        Server that accept a new connections and establishes a
        communication with a client. After client with communication
        finishes, servers starts to accept new connections again
        """
        while self._alive:
            try:
                client_sock = self.__accept_new_connection()
                Server.__handle_client_connection(client_sock)
            except socket.timeout:
                continue
            except OSError as e:
                if self._alive:
                    logging.error(f"action: accept_connection | result: fail | error: {e}")
        self._cleanup()

    @staticmethod
    def __handle_client_connection(client_sock):
        """
        Read message from a specific client socket and closes the socket

        If a problem arises in the communication with the client, the
        client socket will also be closed
        """
        try:
            # TODO: Modify the receive to avoid short-reads
            msg = client_sock.recv(1024).rstrip().decode('utf-8')
            addr = client_sock.getpeername()
            logging.info(f'action: receive_message | result: success | ip: {addr[0]} | msg: {msg}')
            # TODO: Modify the send to avoid short-writes
            client_sock.send("{}\n".format(msg).encode('utf-8'))
        except OSError as e:
            logging.error(f"action: receive_message | result: fail | error: {e}")
        finally:
            client_sock.close()

    def __accept_new_connection(self):
        """
        Accept new connections

        Function blocks until a connection to a client is made.
        Then connection created is printed and returned
        """

        # Connection arrived
        logging.info('action: accept_connections | result: in_progress')
        c, addr = self._server_socket.accept()
        logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
        return c

    def _cleanup(self):
        try:
            self._server_socket.close()
            logging.info('action: graceful_shutdown | result: success')
        except OSError as e:
            logging.error(f'action: server_socket_shutdown | result: fail | error: {e}')

    def shutdown(self):
        logging.info('action: graceful_shutdown | result: in_progress')
        self._alive = False
