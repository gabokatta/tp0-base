import sys
import yaml
import os

PROJECT_NAME = "tp0"
SERVER_SERVICE = "server"
SERVICES_FIELD = "services"
NETWORK_NAME = "tp0_net"
NETWORK_SUBNET = "172.25.125.0/24"
SERVER_BASE_PATH = "./server"
CLIENT_BASE_PATH = "./client"


class CoolDumper(yaml.Dumper):
    def write_line_break(self, data=None):
        super().write_line_break(data)
        if len(self.indents) <= 2:
            super().write_line_break()


def error_exit(msg: str):
    print(f"\nERROR: {msg}")
    sys.exit(1)


def sanitize_filename(file: str) -> str:
    if len(file) == 0:
        error_exit("filename cannot be an empty string")

    if not file.endswith(".yaml"):
        error_exit("invalid file extension, dockerfile must end with .yaml")

    return file


def sanitize_clients(n: str) -> int:
    if not n.isnumeric():
        error_exit("invalid client number, must be an integer")

    parsed = int(n)

    if parsed < 1:
        error_exit("invalid number of clients, must be greater than 0")

    return parsed


def generate_docker_compose(file: str, clients: int):
    compose = base_compose()
    for i in range(1, clients + 1):
        client_name = f"client{i}"
        client = base_client(client_name, i)
        compose[SERVICES_FIELD][client_name] = client

    print(f"generating docker-compose file for {clients} clients at {file}")

    with open(file, "w") as f:
        yaml.dump(compose, f, default_flow_style=False, sort_keys=False, Dumper=CoolDumper)


def base_server():
    return {
        "container_name": SERVER_SERVICE,
        "image": "server:latest",
        "entrypoint": "python3 /main.py",
        "environment": [
            "PYTHONUNBUFFERED=1"
        ],
        "volumes": [
            f"{os.path.abspath(SERVER_BASE_PATH)}/config.ini:/config.ini",
        ],
        "networks": [NETWORK_NAME]
    }


def base_client(name: str, client_id: int):
    return {
        "container_name": name,
        "image": "client:latest",
        "entrypoint": "/client",
        "environment": [
            f"CLI_ID={client_id}"
        ],
        "volumes": [
            f"{os.path.abspath(CLIENT_BASE_PATH)}/config.yaml:/config.yaml",
        ],
        "networks": [NETWORK_NAME],
        "depends_on": [SERVER_SERVICE]
    }


def base_compose():
    return {
        "name": PROJECT_NAME,
        "services": {
            SERVER_SERVICE: base_server()
        },
        "networks": {
            NETWORK_NAME: {
                "name": NETWORK_NAME,
                "ipam": {
                    "driver": "default",
                    "config": [
                        {
                            "subnet": NETWORK_SUBNET
                        }
                    ]
                }
            }
        }
    }


if __name__ == "__main__":
    if len(sys.argv) < 3:
        error_exit("not enough arguments, usage: build_compose.py <filename> <client_n>")

    filename = sanitize_filename(sys.argv[1])
    client_n = sanitize_clients(sys.argv[2])
    generate_docker_compose(filename, client_n)

    sys.exit(0)
