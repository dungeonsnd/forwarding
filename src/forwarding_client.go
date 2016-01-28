package main

import (
	"flag"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

var (
	server0 = flag.String("server0", "127.0.0.1:8601", "Adress connecting to the first server.")
	server1 = flag.String("server1", "127.0.0.1:8080", "Adress connecting to the second server.")
)

type ConnInfo struct {
	conn      net.Conn // tcp连接
	connected bool     // 已经连接
}

var cmdQ = make(chan string) // 控制命令

func main() {
	flag.Parse()
	log.Printf("-- Main started. Server0 addr:%s,  Server1 addr:%s \n", *server0, *server1)

	server0Conn := &ConnInfo{nil, false}
	server1Conn := &ConnInfo{nil, false}

	go ClientProc(*server0, server0Conn, server1Conn)
	go ClientProc(*server1, server1Conn, server0Conn)

	select {
	case cmd := <-cmdQ: // 收到控制指令
		if strings.EqualFold(cmd, "quit") {
			log.Println("quit")
			break
		}
	}
}

func ClientProc(serverAddr string, thisConn *ConnInfo, otherConn *ConnInfo) {
	log.Printf("ClientProc, serverAddr=%s \n", serverAddr)

	for {
		if thisConn.connected == false {
			conn, err := net.Dial("tcp", serverAddr)
			if err != nil {
				log.Printf("serverAddr=%s, connect failed! \n", serverAddr)
				thisConn.connected = false
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			thisConn.conn = conn
			thisConn.connected = true
			log.Printf("serverAddr=%s,  connected, thisConn.conn=%+v, otherConn.conn=%+v",
				serverAddr, thisConn.conn, otherConn.conn)
		}

		for {
			if thisConn.connected == false {
				log.Printf("serverAddr=%s,  thisConn.connected == false, Sleep \n", serverAddr)
				time.Sleep(200 * time.Millisecond)
				break
			}
			if otherConn.conn == nil {
				log.Printf("serverAddr=%s,  otherConn.conn == nil , Sleep \n", serverAddr)
				time.Sleep(200 * time.Millisecond)
				continue
			}

			// 从一个tcp连接读取数据，写入另一个tcp连接。另一协程也是做同样的事情，不过连接正好相反。
			data := make([]byte, 1048576)
			n, err := thisConn.conn.Read(data)
			if err != nil {
				if err == io.EOF {
					log.Println("thisConn closed!")
				} else {
					log.Println("Read error: ", err)
				}
				thisConn.connected = false
				break
			}
			log.Println("Read data:", string(data[:n]))
			otherConn.conn.Write(data[:n])
		}

	}
}
