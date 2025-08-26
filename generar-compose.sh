#!/bin/bash
pip install -r requirements.txt
printf "\nDOCKER COMPOSE BUILDER\n"
python3 ./scripts/build_compose.py "$1" "$2"