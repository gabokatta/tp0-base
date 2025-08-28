import logging

from server import Server


class Shutdown:
    """
    Simple shutdown flag for the server.
    Handles SIGTERM & SIGINT to trigger the graceful shutdown.
    """

    def __init__(self, server: Server):
        self._triggered = False
        self._server = server

    def trigger(self, signum, frame):
        logging.info(f'action: signal_received | code: {signum}')
        self._server.shutdown()
