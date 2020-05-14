#!/bin/bash

cd protocol/blaster/pb
protoc --go_out=. --go_opt=paths=source_relative wire.proto
