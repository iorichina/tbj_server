package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
)

// go build tcp_server.go
//
// vi run-server.sh
//
// ./tcp_server >>server.log 2>&1 &
//
// sudo sh run-server.sh
//
// TCP Server端测试
// 处理函数
func main() {
	logger := log.New(os.Stdout, "", log.Lmsgprefix|log.Ldate|log.Lmicroseconds)
	listen, err := net.Listen("tcp", "0.0.0.0:80")
	if err != nil {
		logger.Printf("Listen() failed, err %#v\n", err)
		return
	}
	for {
		conn, err := listen.Accept() // 监听客户端的连接请求
		if err != nil {
			logger.Printf("Accept() failed, err: %#v\n", err)
			continue
		}
		go process(conn) // 启动一个goroutine来处理客户端的连接请求
	}
}

func process(conn net.Conn) {
	remote := conn.RemoteAddr()
	var mac string
	var err error
	logger := log.New(os.Stdout, fmt.Sprintf("[%17v][%v]", mac, remote), log.Lmsgprefix|log.Ldate|log.Lmicroseconds)

	defer func() {
		if r := recover(); r != nil {
			logger.Printf("Recovered from panic: %#v", r)
		}
	}()

	defer func() {
		err = conn.Close()
		if err != nil {
			logger.Printf("Connection Close err %v\n", err)
			return
		}
		logger.Printf("Connection Close\n")
	}() // 关闭连接
	logger.Printf("Connection Connected to %v\n", conn.LocalAddr())

	_, err = conn.Write(newServerMsg(0x30, []byte{90, 0}))
	logger.Printf("Write(0x30) by connected with %v\n", err)
	_, err = conn.Write(newServerMsg(0x33, nil))
	logger.Printf("Write(0x33) by connected with %v\n", err)
	_, err = conn.Write(newServerMsg(0x42, nil))
	logger.Printf("Write(0x42) by connected with %v\n", err)
	_, err = conn.Write(newServerMsg(0x34, nil))
	logger.Printf("Write(0x34) by connected with %v\n", err)

	var n int
	reader := bufio.NewReader(conn)
	var buf [512]byte
	for {
		n, err = reader.Read(buf[:]) // 读取数据
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Printf("Read EOF %v\n", err)
			} else {
				logger.Printf("Read failed err %v\n", err)
			}
			break
		}

		if n > 8 && 0xFE == buf[0] && 0x01 == buf[3] {
			if false && 0x14 == buf[7] {
				_, err = conn.Write(buf[:n])
				if err != nil {
					logger.Printf("Reply 0x14 err %v\n", err)
					continue
				} // 发送数据
				continue
			}
			if 0x31 == buf[7] {
				logger.Printf("Read %#x(%d) with % X\n", buf[7], int(buf[1])*256+int(buf[2]), buf[8:10])
				continue
			}
			if 0x34 == buf[7] {
				mac = strings.Join([]string{string(buf[9:11]), string(buf[11:13]), string(buf[13:15]), string(buf[15:17]), string(buf[17:19]), string(buf[19:21])}, ":")
				logger.Printf("Read %#x(%d) with %d (0=idle,1=playing,>1=error)\n", buf[7], int(buf[1])*256+int(buf[2]), buf[8])
				if buf[8] == 0 {
					_, err = conn.Write(newServerMsg(0x30, []byte{90, 0}))
					logger.Printf("Write(0x30) with %v\n", err)
				}
				continue
			}
			if 0x35 == buf[7] {
				m := strings.Join([]string{string(buf[8:10]), string(buf[10:12]), string(buf[12:14]), string(buf[14:16]), string(buf[16:18]), string(buf[18:20])}, ":")
				if m != mac {
					logger.SetPrefix(fmt.Sprintf("[%17v][%v]", m, remote))
				}
				mac = m

				_, err = conn.Write(newServerMsg(0x33, nil))
				logger.Printf("Write(0x33) with %v\n", remote, mac, err)
				_, err = conn.Write(newServerMsg(0x31, []byte{0, 1}))
				logger.Printf("Write(0x31) with %v\n", remote, mac, err)
				_, err = conn.Write(newServerMsg(0x32, []byte{0x02}))
				logger.Printf("Write(0x32) with %v\n", remote, mac, err)
				_, err = conn.Write(newServerMsg(0x34, nil))
				logger.Printf("Write(0x34) with %v\n", err)
				continue
			}
			continue
		}
	}
}

func newServerMsg(cmd byte, data []byte) []byte {
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
