#!/bin/bash

cd blaster/pb
protoc --go_out=. --go_opt=paths=source_relative wire.proto
