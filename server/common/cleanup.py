import logging

from common.server import Server


class Shutdown:
    """
    Simple shutdown flag for the server.
    Handles SIGTERM & SIGINT to trigger the graceful shutdown.
    """

    def __init__(self, server: Server):
        self._triggered = False
        self._server = server

    def trigger(self, signum, frame):
        logging.debug(f'action: signal_received | result: in_progress | code: {signum}')
        self._server.shutdown()
