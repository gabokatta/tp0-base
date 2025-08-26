#!/bin/bash
printf "\nDOCKER COMPOSE BUILDER\n"
python3 ./scripts/build_compose.py "$1" "$2"