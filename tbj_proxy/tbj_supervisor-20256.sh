#!/bin/sh

while true; do
	server=`ps -ef | grep tbj_proxy | grep "20256" | grep -v grep`
	if [ ! "$server" ]; then
		./tbj_proxy 0.0.0.0:20256 3000  127.0.0.1:20009 3000 >>log-tbj-20256.log 2>&1 &
		sleep 6
	fi
	sleep 1
done
