#!/bin/bash

docker run --name=grafana --net=host -d -p 3000:3000 grafana/grafana
