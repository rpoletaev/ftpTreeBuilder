#!/bin/bash

CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ../../../../gitlab.com/garanteka/goszakupki/docker-nmp/ftpTreeBuilder/ftpTreeBuilder .