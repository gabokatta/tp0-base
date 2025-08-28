#!/bin/bash

set -e

if [ ! -f "requirements.txt" ]; then
    printf "\nError: requirements.txt not found"
    exit 1
fi

if [ ! -d "venv" ]; then
    printf "creating venv...\n"
    python3 -m venv venv
fi

source venv/bin/activate

pip install -r requirements.txt > /dev/null 2>&1