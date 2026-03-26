#!/bin/bash
set -e
go build -o doclist . && ./doclist --dump --depth=3 7bff278ed5b795a1f074acb5 reframe
