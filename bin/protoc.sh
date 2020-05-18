#!/bin/bash

if [[ $# -ne 1 ]]; then
	echo "usage: protoc.sh <pb_dir>"
	exit 1
fi

cd $1
protoc --go_out=. --go_opt=paths=source_relative wire.proto
