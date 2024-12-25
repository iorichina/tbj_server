#!/bin/sh

go build tcp_client_middleware1.go

while true; do
        server=`ps -ef | grep tcp_client_middleware1 | grep "192.168.1.38:2038" | grep -v grep`
        if [ ! "$server" ]; then
            ./tcp_client_middleware1 192.168.1.38:2038 3000  140.143.165.157:10112 10000 >>log-2038.log 2>&1 &
            sleep 10
        fi
        sleep 1
done