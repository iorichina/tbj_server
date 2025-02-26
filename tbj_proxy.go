package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) <= 4 {
		log.Fatalf("usage: tbj_proxy listen-ip-port tbj-timeout-millis server-ip-port server-timeout-millis")
	}
	logger := log.New(os.Stdout, "main   ", log.Lmsgprefix|log.Ldate|log.Lmicroseconds)

	listenAddr := os.Args[1]
	tbjTimeout, err := strconv.ParseInt(os.Args[2], 10, 64)
	if err != nil {
		logger.Fatalf("parse tbj-timeout-millis fail terminate %s", err)
	}
	serverAddr := os.Args[3]
	serverTimeout, err := strconv.ParseInt(os.Args[4], 10, 64)
	if err != nil {
		logger.Fatalf("parse server-timeout-millis fail terminate %s", err)
	}

	tbjProxy := &TbjProxy{
		listenAddr:    listenAddr,
		tbjTimeout:    time.Duration(tbjTimeout) * time.Millisecond,
		serverAddr:    serverAddr,
		serverTimeout: time.Duration(serverTimeout) * time.Millisecond,
	}
	tbjProxy.tcpServer()
}

type TbjProxy struct {
	listenAddr    string
	tbjTimeout    time.Duration
	serverAddr    string
	serverTimeout time.Duration
}

func (m *TbjProxy) tcpServer() {
	logger := log.New(os.Stdout, "tcpServer ", log.Lmsgprefix|log.Ldate|log.Lmicroseconds)
	listen, err := net.Listen("tcp", m.listenAddr) // 监听端口
	if err != nil {
		logger.Printf("Listen to %v failed terminate %#v\n", m.listenAddr, err)
		return
	}
	for {
		conn, err := listen.Accept() // 监听tbj的连接请求
		if err != nil {
			logger.Printf("%v Accept() failed, err: %v\t%#v\n", m.listenAddr, err.Error(), err)
			continue
		}
		go m.serverProcess(conn) // 启动一个goroutine来处理tbj的连接请求
	}
}

func (m *TbjProxy) serverProcess(tbjConn net.Conn) {
	logger := log.New(os.Stdout, "serverProcess ", log.Lmsgprefix|log.Ldate|log.Lmicroseconds)
	var mac string

	defer func() {
		if r := recover(); r != nil {
			logger.Printf("[%v][%v]Recovered from panic: %#v", mac, tbjConn.RemoteAddr(), r)
		}
	}()
	defer func(tbjConn net.Conn) {
		err := tbjConn.Close()
		if err != nil {
			logger.Printf("[%v][%v]tbjConn Close err %v\t%#v\n", mac, tbjConn.RemoteAddr(), err.Error(), err)
		}
	}(tbjConn) // 关闭连接
	logger.Printf("[%v]Connected by tbj:%v  at serverAddr:%v\n", mac, tbjConn.RemoteAddr(), tbjConn.LocalAddr())

	var err error
	client, err := tcpClient()
	if err != nil {
		logger.Printf("%v[%v]client Connect to remote failed, err: %v\t%#v\n", time.Now().Format("2006-01-02 15:04:05.000"), tbjConn.RemoteAddr(), err.Error(), err)
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			now := time.Now().Format("2006-01-02 15:04:05.000")
			logger.Printf("%v[%v][%v]client Close err %v\t%#v\n", now, conn.RemoteAddr(), mac, err.Error(), err)
		}
	}(client) // 关闭连接
	now = time.Now().Format("2006-01-02 15:04:05.000")
	logger.Printf("%v[%v]Connected by %v\n", now, client.RemoteAddr(), client.LocalAddr())

	_, err = tbjConn.Write([]byte{254, 134, 226, 1, 121, 29, 9, 52, 61})
	now = time.Now().Format("2006-01-02 15:04:05.000")
	if err != nil {
		logger.Printf("%v[%v]server Write(0x34) err %v\n", now, tbjConn.RemoteAddr(), err)
		return
	}

	closeChan := make(chan error)
	defer close(closeChan)
	macChan := make(chan string)
	defer close(macChan)
	go func(macChan chan string) {
		for m := range macChan {
			mac = m
		}
	}(macChan)

	go handleServer(logger, macChan, tbjConn, client, closeChan)
	go handleClient(logger, macChan, tbjConn, client, closeChan)
	err = <-closeChan
	logger.Printf("%v[%v]server[%v] or client[%v] err %v\t%#v disconnect\n", now, mac, tbjConn.RemoteAddr(), client.RemoteAddr(), err.Error(), err)
}

func tcpClient() (net.Conn, error) {
	return net.Dial("tcp", "406a7637n7.goho.co:11180")
}

func handleClient(logger *log.Logger, macChan chan string, conn net.Conn, client net.Conn, closeChan chan error) {
	remote := client.RemoteAddr()
	now := time.Now().Format("2006-01-02 15:04:05.000")
	var mac string
	go func(macChan chan string) {
		for m := range macChan {
			mac = m
		}
	}(macChan)
	var err error
	var n int
	reader := bufio.NewReader(client)
	var buf [512]byte
	for {
		now = time.Now().Format("2006-01-02 15:04:05.000")
		n, err = reader.Read(buf[:]) // 读取数据
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Printf("%v[%v][%v]client Read EOF %#v\n", now, remote, mac, err)
			} else {
				logger.Printf("%v[%v][%v]client Read failed err %v\t%#v\n", now, remote, mac, err.Error(), err)
			}
			closeChan <- err
			break
		}

		now = time.Now().Format("2006-01-02 15:04:05.000")
		if n > 8 && 0xFE == buf[0] && 0x01 == buf[3] {
			if 0x31 == buf[7] {
				logger.Printf("%v[%v][%v]client 读 %#x\t% X\n", now, remote, mac, buf[7], buf[:n])
			}
		}

		_, err = conn.Write(buf[:n])
		if err != nil {
			now = time.Now().Format("2006-01-02 15:04:05.000")
			logger.Printf("%v[%v][%v]server Write err %v\t%#v\n", now, conn.RemoteAddr(), mac, err.Error(), err)
			closeChan <- err
			break
		} // 发送数据
	}
}

func handleServer(logger *log.Logger, macChan chan string, conn net.Conn, client net.Conn, closeChan chan error) {
	remote := conn.RemoteAddr()
	now := time.Now().Format("2006-01-02 15:04:05.000")
	var mac string
	var err error
	var n int
	reader := bufio.NewReader(conn)
	var buf [512]byte
	for {
		n, err = reader.Read(buf[:]) // 读取数据
		if err != nil {
			now = time.Now().Format("2006-01-02 15:04:05.000")
			if errors.Is(err, io.EOF) {
				logger.Printf("%v[%v][%v]server Read EOF %#v\n", now, remote, mac, err)
			} else {
				logger.Printf("%v[%v][%v]server Read failed err %v\t%#v\n", now, remote, mac, err.Error(), err)
			}
			closeChan <- err
			break
		}

		now = time.Now().Format("2006-01-02 15:04:05.000")
		if n > 8 && 0xFE == buf[0] && 0x01 == buf[3] {
			if 0x34 == buf[7] {
				mac = strings.Join([]string{string(buf[9:11]), string(buf[11:13]), string(buf[13:15]), string(buf[15:17]), string(buf[17:19]), string(buf[19:21])}, ":")
				macChan <- mac
			}
			if 0x35 == buf[7] {
				mac = strings.Join([]string{string(buf[8:10]), string(buf[10:12]), string(buf[12:14]), string(buf[14:16]), string(buf[16:18]), string(buf[18:20])}, ":")
				logger.Printf("%v[%v][%v]server 读 %#x 心跳\n", now, remote, mac, buf[7])
				macChan <- mac
			}
			if 0x30 == buf[7] || 0x3b == buf[7] {
				logger.Printf("%v[%v][%v]server 读 %#x\t% X\n", now, remote, mac, buf[7], buf[:n])
			}
		}

		_, err = client.Write(buf[:n])
		if err != nil {
			now = time.Now().Format("2006-01-02 15:04:05.000")
			logger.Printf("%v[%v][%v]client Write err %v\t%#v\n", now, client.RemoteAddr(), mac, err.Error(), err)
			closeChan <- err
			break
		} // 发送数据
	}
}
