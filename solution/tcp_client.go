package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// TCP 客户端
func main() {
	now := time.Now().Format("2006-01-02 15:04:05.000")
	f, err := os.OpenFile("client.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend|os.ModePerm)
	if err != nil {
		log.Fatalf("create file client.log failed: %v", err)
	}
	logger := log.New(io.MultiWriter(os.Stdout, f), "", 0)

	targetAddr := "40676377mc.vicp.fun:11180"
	client, err := net.Dial("tcp", targetAddr)
	if err != nil {
		logger.Printf("%v client Dial(%v) failed, err %v\t%#v\n", time.Now().Format("2006-01-02 15:04:05.000"), targetAddr, err.Error(), err)
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			logger.Printf("%v[%v]client Close() err %v\t%#v\n", time.Now().Format("2006-01-02 15:04:05.000"), conn.RemoteAddr(), err.Error(), err)
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

		inputChan <- ss
	}(inputReader, inputChan)

	// reader := bufio.NewReader(client)
	// buf := [512]byte{}
	bb := []byte{254, 134, 226, 1, 121, 29, 9, 52, 61}
	var tb []byte = nil
	for {
		if nil == tb {
			tb = bb
		}
		_, err = client.Write(tb) // 发送数据
		if err != nil {
			logger.Printf("%v[%v]client Write(0x34) err %v\n", now, remote, err)
			return
		}
		tb = nil

		// _, err = reader.Read(buf[:])
		// now = time.Now().Format("2006-01-02 15:04:05.000")
		// if err != nil {
		// 	if errors.Is(err, io.EOF) {
		// 		logger.Printf("%v[%v]client Read EOF %#v\n", now, remote, err)
		// 	} else {
		// 		logger.Printf("%v[%v]client Read failed err %v\t%#v\n", now, remote, err.Error(), err)
		// 	}
		// 	return
		// }

		select {
		case ss, ok := <-inputChan:
			if !ok {
				break
			}
			tb = ss
		case <-time.After(10 * time.Second):
		}
	}
}
