#!/bin/bash

docker run -d -p 8086:8086 --name=influxdb -v ~/local/opt/influxdb/var/lib/influxdb:/var/lib/influxdb influxdb:1.8.3
