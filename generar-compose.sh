#!/bin/bash
bash ./scripts/py_env_check.sh
printf "\nDOCKER COMPOSE BUILDER\n"
python3 ./scripts/build_compose.py "$1" "$2"