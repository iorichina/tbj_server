package main

import (
	"bufio"
	"errors"
	"fmt"
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
	f, err := os.OpenFile("client.log", os.O_APPEND|os.O_WRONLY|os.O_CREATE, os.ModeAppend|os.ModePerm)
	if err != nil {
		log.Fatalf("create file client.log failed: %v", err)
	}
	logger := log.New(io.MultiWriter(os.Stdout, f), "", 0)

	client, err := net.Dial("tcp", "127.0.0.1:8880")
	if err != nil {
		fmt.Printf("%v Dial() failed, err %#v\n", time.Now().Format("2006-01-02 15:04:05.000"), err)
		return
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			fmt.Printf("%v[%v] Close() err %v\t%#v\n", time.Now().Format("2006-01-02 15:04:05.000"), conn.RemoteAddr(), err.Error(), err)
		}
	}(client) // 关闭TCP连接

	remote := client.RemoteAddr()
	inputReader := bufio.NewReader(os.Stdin)
	for {
		input, _ := inputReader.ReadString('\n') // 读取用户输入
		inputInfo := strings.Trim(input, "\r\n")
		if strings.ToUpper(inputInfo) == "Q" { // 如果输入q就退出
			return
		}

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
		_, err := client.Write(ss) // 发送数据
		if err != nil {
			fmt.Printf("%v[%v]Write(%v) err %#v\n", time.Now().Format("2006-01-02 15:04:05.000"), remote, inputInfo, err)
			return
		}

		buf := [512]byte{}
		n, err := client.Read(buf[:])
		now := time.Now().Format("2006-01-02 15:04:05.000")
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Printf("%v[%v]client Read EOF %#v\n", now, remote, err)
			} else {
				logger.Printf("%v[%v]client Read failed err %v\t%#v\n", now, remote, err.Error(), err)
			}
			return
		}
		logger.Println(string(buf[:n]))
	}
}
