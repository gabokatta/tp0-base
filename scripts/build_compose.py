import os
import sys
import yaml
import ipaddress


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
    print(f"\ngenerating docker-compose for {clients} clients in {file}")


if __name__ == "__main__":
    if len(sys.argv) < 3:
        error_exit("not enough arguments, usage: build_compose.py <filename> <client_n>")

    filename = sanitize_filename(sys.argv[1])
    client_n = sanitize_clients(sys.argv[2])
    generate_docker_compose(filename, client_n)

    sys.exit(0)
