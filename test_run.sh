#!/bin/bash
set -e
go build -o doclist .
# ./doclist --dump --depth=3 7bff278ed5b795a1f074acb5 reframe
./doclist --dump --root-type=resourcecompanyowner --secrets=personal/secrets.json 5a998a2b5618f7127b13db54 furious
