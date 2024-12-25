package main

import (
	"bufio"
	"io"
	"log"
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// ./tcp_client_as_server host/ip:port tcp_client_as_server_xx.log
func main() {
	f, err := os.OpenFile(os.Args[2], os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend|os.ModePerm)
	now := time.Now().Format("2006-01-02 15:04:05.000")
	if err != nil {
		log.Fatalf("create file %v failed: %v", os.Args[2], err)
	}
	logger := log.New(io.MultiWriter(os.Stdout, f), "", 0)

	client, err := net.Dial("tcp", os.Args[1])
	if err != nil {
		logger.Printf("%v Dial(%v) failed, err %v\t%#v\n", time.Now().Format("2006-01-02 15:04:05.000"), os.Args[1], err.Error(), err)
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Printf("%v[%v]Close() err %v\t%#v\n", time.Now().Format("2006-01-02 15:04:05.000"), conn.RemoteAddr(), err.Error(), err)
		}
	}(client) // 关闭TCP连接
	now = time.Now().Format("2006-01-02 15:04:05.000")
	remote := client.RemoteAddr()
	logger.Printf("%v[%v]Connected by %v\n", now, remote, client.LocalAddr())

	inputReader := bufio.NewReader(os.Stdin)
	inputChan := make(chan []byte)
	go func(reader *bufio.Reader, inputChan chan []byte) {
		input, _ := inputReader.ReadString('\n') // 读取用户输入
		inputInfo := strings.Trim(input, "\r\n")
		//if strings.ToUpper(inputInfo) == "Q" { // 如果输入q就退出
		//	return
		//}

		var ss []byte
		//fe 86 e2 01 79 1d 0b 31 00 01 3d
		//fe 49 42 01 b6 bd 0b 31 00 01 3d
		//fe 86 e2 01 79 1d 0a 32 02 3e
		if 'f' == input[0] && 'e' == input[1] && ' ' == input[2] && len(strings.Split(inputInfo, " ")) > 8 {
			ss = make([]byte, 0)
			split := strings.Split(inputInfo, " ")
			for _, s := range split {
				i, _ := strconv.ParseInt(s, 16, 16)
				ss = append(ss, byte(i))
			}
		} else {
			ss = []byte(inputInfo)
		}

		_, err = client.Write(ss) // 发送数据
		if err != nil {
			logger.Printf("%v[%v]Write err %v\n", now, remote, err)
			return
		}
	}(inputReader, inputChan)

	reader := bufio.NewReader(client)
	buf := [512]byte{}
	var n int
	var mac string
	for {
		n, err = reader.Read(buf[:])
		if err != nil {
			now = time.Now().Format("2006-01-02 15:04:05.000")
			if errors.Is(err, io.EOF) {
				logger.Printf("%v[%v]Read EOF %#v\n", now, remote, err)
			} else {
				logger.Printf("%v[%v]Read failed err %v\t%#v\n", now, remote, err.Error(), err)
			}
			return
		}
		if n > 8 && 0xFE == buf[0] && 0x01 == buf[3] {
			if 0x34 != buf[7] && 0x35 != buf[7] {
				continue
			}
			if 0x34 == buf[7] {
				mac = strings.Join([]string{string(buf[9:11]), string(buf[11:13]), string(buf[13:15]), string(buf[15:17]), string(buf[17:19]), string(buf[19:21])}, ":")
				logger.Printf("%v[%v][%v]读 %#x 状态 %d (0=idle,1=playing,>1=error)\n", now, remote, mac, buf[7], buf[8])
				continue
			}
			if 0x35 == buf[7] {
				mac = strings.Join([]string{string(buf[8:10]), string(buf[10:12]), string(buf[12:14]), string(buf[14:16]), string(buf[16:18]), string(buf[18:20])}, ":")
				logger.Printf("%v[%v][%v]读 %#x 心跳\n", now, remote, mac, buf[7])
				continue
			}
			logger.Printf("%v[%v][%v]读 %#x\t% X\n", now, remote, mac, buf[7], buf[:n])
			continue
		}
		logger.Printf("%v[%v]Read % X\n", now, remote, buf[:n])
	}
}
