#!/bin/sh

go build tbj_proxy.go

while true; do
	server=`ps -ef | grep tbj_proxy | grep "192.168.1.101:2102" | grep -v grep`
	if [ ! "$server" ]; then
		./tbj_proxy 0.0.0.0:20251 3000  8.138.173.89:20009 3000 >>log-20251.log 2>&1 &
		sleep 5
	fi
	sleep 1
done
