#!/bin/sh

go build tcp_client_middleware1.go

while true; do
	server=`ps -ef | grep tcp_client_middleware1 | grep "192.168.1.101:2102" | grep -v grep`
	if [ ! "$server" ]; then
		./tcp_client_middleware1 192.168.1.101:2102 3000  coinpush.pokekara.com:20112 3000 >>log-101.log 2>&1 &
		sleep 5
	fi
	sleep 1
done
