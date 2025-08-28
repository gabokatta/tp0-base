import subprocess
import sys

SUCCESS = "success"
FAIL = "fail"
ACTION = "test_echo_server"
NETWORK = "tp0_testing_net"

SERVER_HOST = "server"
SERVER_PORT = "12345"

MESSAGE = "ITS TV TIME"
NETCAT_COMMAND = f'echo "{MESSAGE}" | nc {SERVER_HOST} {SERVER_PORT}'
DOCKER_COMMAND = f"docker run --rm --network {NETWORK} busybox:latest /bin/sh -c '{NETCAT_COMMAND}'"


def msg_template(status: str):
    return f"action: {ACTION} | result: {status}"


if __name__ == "__main__":
    try:
        result = subprocess.run(DOCKER_COMMAND, shell=True, capture_output=True, text=True)

        response = result.stdout.strip()
        print(response)
        print(msg_template(SUCCESS)) if response == MESSAGE else print(msg_template(FAIL))
        sys.exit(0)
    except Exception as e:
        print(e)
        sys.exit(1)
