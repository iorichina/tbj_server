package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type TbjProxy struct {
	listenAddr    string
	tbjTimeout    time.Duration
	serverAddr    string
	serverTimeout time.Duration
}

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

func (m *TbjProxy) tcpServer() {
	logger := log.New(os.Stdout, "[tcpServer]", log.Lmsgprefix|log.Ldate|log.Lmicroseconds)
	listen, err := net.Listen("tcp", m.listenAddr) // 监听端口
	if err != nil {
		logger.Printf("Listen to %v failed terminate %v\n", m.listenAddr, err)
		return
	}
	logger.Printf("Listened to %v \n", m.listenAddr)
	for {
		conn, err := listen.Accept() // 监听tbj的连接请求
		if err != nil {
			logger.Printf("%v Accept() failed, err: %v\n", m.listenAddr, err)
			continue
		}
		go m.serverProcess(conn) // 启动一个goroutine来处理tbj的连接请求
	}
}

func (m *TbjProxy) serverProcess(tbjConn net.Conn) {
	var mac string
	logger := log.New(os.Stdout, fmt.Sprintf("[%17v]main([%v]->[%v]) ", mac, tbjConn.RemoteAddr(), m.serverAddr), log.Lmsgprefix|log.Ldate|log.Lmicroseconds)

	defer func() {
		if r := recover(); r != nil {
			logger.Printf("Recovered from panic: err in serverProcess %v", r)
		}
	}() //panic处理

	defer func(tbjConn net.Conn) {
		err := tbjConn.Close()
		if err != nil {
			logger.Printf("tbjConn Close err %v\n", err)
		} else {
			logger.Printf("tbjConn Closed\n")
		}
	}(tbjConn) // 关闭连接
	logger.Printf("tbj Connected to local:%v\n", tbjConn.LocalAddr())

	// 发起服务器的连接
	serverConn, err := net.DialTimeout("tcp", m.serverAddr, m.serverTimeout)
	if nil != err {
		logger.Printf("server-ip-port unreachable err %v", err)
		return
	}
	defer func(serverConn net.Conn) {
		err := serverConn.Close()
		if err != nil {
			logger.Printf("serverConn Close err %v\n", err)
		} else {
			logger.Printf("serverConn Closed\n")
		}
	}(serverConn) // 关闭连接
	logger.SetPrefix(fmt.Sprintf("[%17v]main([%v]->[%v]) ", mac, tbjConn.RemoteAddr(), serverConn.RemoteAddr()))
	logger.Printf("Connected to server by local:%v\n", serverConn.LocalAddr())

	//tbj或server连接任意一个报错都断开两个连接，让tbj自动重连
	tbjErrChan := make(chan error)
	serverErrChan := make(chan error)

	//用于日志增加mac
	macChan := make(chan string, 128)
	macChanTbj := make(chan string, 128)
	macChanServer := make(chan string, 128)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Printf("Recovered from panic: err in macChan range %v", r)
			}
		}()
		for m := range macChan {
			if m != mac {
				logger.SetPrefix(fmt.Sprintf("[%17v]main([%v]->[%v]) ", m, tbjConn.RemoteAddr(), serverConn.RemoteAddr()))
			}
			mac = m
			macChanTbj <- m
			macChanServer <- m
		}
		defer close(macChanTbj)
		defer close(macChanServer)
	}()

	go m.handleTbj(macChan, macChanTbj, tbjConn, serverConn, tbjErrChan)
	go m.handleServer(macChan, macChanServer, tbjConn, serverConn, serverErrChan)

	select {
	case err = <-tbjErrChan:
		logger.Printf("Connection tbj err %v, disconnect all\n", err)
		return
	case err = <-serverErrChan:
		logger.Printf("Connection server err %v, disconnect all\n", err)
		return
	}
}

func (m *TbjProxy) handleTbj(macChan, macChanTbj chan string, tbjConn, serverConn net.Conn, tbjErrChan chan error) {
	var mac string
	logger := log.New(os.Stdout, fmt.Sprintf("[%17v]tbj [%v]  ", mac, tbjConn.RemoteAddr()), log.Lmsgprefix|log.Ldate|log.Lmicroseconds)

	defer func() {
		if r := recover(); r != nil {
			logger.Printf("Recovered from panic: err in handleTbj %v", r)
		}
	}() //panic处理
	defer close(tbjErrChan)
	defer close(macChan)

	//日志增加mac
	go func() {
		for m := range macChanTbj {
			if m != mac {
				logger.SetPrefix(fmt.Sprintf("[%17v]tbj [%v]  ", m, tbjConn.RemoteAddr()))
			}
			mac = m
		}
	}()

	//5、	查询机器状态（指令码0x34)
	_, err := tbjConn.Write(m.newMsg(0x34, nil))
	logger.Printf("Write(0x34) by connected with %v\n", err)

	reader := tbjConn //bufio.NewReader(tbjConn)
	var buf [1024]byte
	var n int
	// 2、	投币命令（指令码0x31） 5、	查询机器状态（指令码0x34) 6、	发送心跳（指令码0x35)
	for {
		n, err = reader.Read(buf[:])
		if err != nil {
			tbjErrChan <- err
			break
		}
		if n <= 0 {
			continue
		}

		logSent := false
		if n > 8 && 0xFE == buf[0] && 0x01 == buf[3] {
			if 0x34 == buf[7] {
				m := strings.Join([]string{string(buf[9:11]), string(buf[11:13]), string(buf[13:15]), string(buf[15:17]), string(buf[17:19]), string(buf[19:21])}, ":")
				if m != mac {
					logger.SetPrefix(fmt.Sprintf("[%17v]tbj [%v]  ", m, tbjConn.RemoteAddr()))
				}
				mac = m
				macChan <- m
				logger.Printf("tbjRead %#v(%d) status %d (0=idle,1=playing,>1=error)\n", buf[7], int(buf[1])*256+int(buf[2]), buf[8])
				logSent = true
			} else if 0x35 == buf[7] {
				m := strings.Join([]string{string(buf[8:10]), string(buf[10:12]), string(buf[12:14]), string(buf[14:16]), string(buf[16:18]), string(buf[18:20])}, ":")
				if m != mac {
					logger.SetPrefix(fmt.Sprintf("[%17v]tbj [%v]  ", m, tbjConn.RemoteAddr()))
				}
				mac = m
				macChan <- m
				logger.Printf("tbjRead %#v(%d) heartbeat\n", buf[7], int(buf[1])*256+int(buf[2]))
				logSent = true
			} else if 0x31 == buf[7] {
				// logger.Printf("Read %#v(%d) insert coin % X\n", buf[7], int(buf[1])*256+int(buf[2]), buf[8:10])
				logger.Printf("tbjRead %#v(%d) insert coin %d\n", buf[7], int(buf[1])*256+int(buf[2]), int(buf[8])*256+int(buf[9]))
				logSent = true
			} else if 0x33 == buf[7] {
				// 4、	查询获得币数（指令码0x33)
				logger.Printf("tbjRead %#v(%d) query coin %d\n", buf[7], int(buf[1])*256+int(buf[2]), int(buf[8])*256+int(buf[9]))
				logSent = true
			} else if 0x14 == buf[7] {
				// 8、	与游戏相关（0x14)
				data := int(buf[8])
				if data == 0 || data == 1 {
					logger.Printf("tbjRead %#v(%d) query coin %d\n", buf[7], int(buf[1])*256+int(buf[2]), int(buf[8]))
					logSent = true
				}
			} else {
				logger.Printf("tbjRead %#v(%d)\n", buf[7], int(buf[1])*256+int(buf[2]))
			}
		}

		dup := make([]byte, n)
		copy(dup, buf[:n])
		n, err = serverConn.Write(dup)
		if logSent {
			logger.Printf("tbjSent %#v(%d) to remote[%v] res %v\n", dup[7], int(buf[1])*256+int(buf[2]), serverConn.RemoteAddr(), err)
		}
	}
}

func (m *TbjProxy) handleServer(macChan, macChanServer chan string, tbjConn, serverConn net.Conn, serverErrChan chan error) {
	var mac string
	logger := log.New(os.Stdout, fmt.Sprintf("[%17v]serv[%v]  ", mac, serverConn.RemoteAddr()), log.Lmsgprefix|log.Ldate|log.Lmicroseconds)

	defer func() {
		if r := recover(); r != nil {
			logger.Printf("Recovered from panic: err in handleServer %v", r)
		}
	}() //panic处理
	defer close(serverErrChan)

	go func() {
		for m := range macChanServer {
			if m != mac {
				logger.SetPrefix(fmt.Sprintf("[%17v]serv[%v]  ", m, serverConn.RemoteAddr()))
			}
			mac = m
		}
	}()

	var err error
	reader := serverConn //bufio.NewReader(serverConn)
	var buf [1024]byte
	var n int
	// 2、	投币命令（指令码0x31） 5、	查询机器状态（指令码0x34) 6、	发送心跳（指令码0x35)
	for {
		n, err = reader.Read(buf[:])
		if err != nil {
			serverErrChan <- err
			break
		}
		if n <= 0 {
			continue
		}

		logSent := false
		if n > 8 && 0xFE == buf[0] && 0x01 == buf[3] {
			if 0x31 == buf[7] {
				// logger.Printf("Read %#v(%d) insert coin % X\n", buf[7], int(buf[1])*256+int(buf[2]), buf[8:10])
				logger.Printf("servRead %#v(%d) insert coin %d\n", buf[7], int(buf[1])*256+int(buf[2]), int(buf[8])*256+int(buf[9]))
				logSent = true
			} else if 0x34 == buf[7] {
				logSent = true
			} else {
				logger.Printf("servRead %#v(%d)\n", buf[7], int(buf[1])*256+int(buf[2]))
			}
		}

		dup := make([]byte, n)
		copy(dup, buf[:n])
		n, err = tbjConn.Write(dup)
		if logSent {
			logger.Printf("servSent %#v(%d) to tbj [%v] res %v\n", dup[7], int(buf[1])*256+int(buf[2]), tbjConn.RemoteAddr(), err)
		}
	}
}

func (m *TbjProxy) newMsg(cmd byte, data []byte) []byte {
	length := 6 + 1 + 1 + len(data) + 1
	msg := make([]byte, length)
	id := rand.Int() & 0xFFFF
	msg[0] = 0xFE
	msg[1] = byte(id >> 8)
	msg[2] = byte(id & 0xFF)
	msg[3] = 0x01
	msg[4] = ^msg[1]
	msg[5] = ^msg[2]
	msg[6] = byte(length)
	msg[7] = cmd
	sum := int(msg[6]) + int(msg[7])
	if nil != data && len(data) > 0 {
		for i, v := range data {
			msg[8+i] = v
			sum += int(v)
		}
	}
	msg[length-1] = byte(sum & 0xFF)
	return msg
}
