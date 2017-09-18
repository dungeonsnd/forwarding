package main

import (
    "net"
    "os"
    "fmt"
    "time"
)

const BUFF_SIZE = 1024
var buff = make([]byte,BUFF_SIZE)

// 接受一个TCPConn处理内容
func handleConn(tcpConn net.Conn){
    if tcpConn == nil {
        return
    }
    for {
        n,err := tcpConn.Read(buff)
        if err != nil {
            fmt.Printf("The RemoteAddr:%s is closed! err:%v \n\n",
                tcpConn.RemoteAddr().String(), err)
            return
        }
        handleError(err)
        if string(buff[:n]) == "exit" {
            fmt.Printf("The client:%s has exited \n\n",tcpConn.RemoteAddr().String())
        }
        if n > 0 {
            fmt.Printf("==>>[%v %v]: %v \n\n",
                tcpConn.RemoteAddr().String(),
                time.Now().Format("2006-01-02 15:04:05"),
                string(buff[:n]))
            tcpConn.Write(buff[:n])
        }
    }
}
// 错误处理
func handleError(err error) {
    if err == nil {
        return
    }
    fmt.Printf("error:%s \n\n",err.Error())
}

func main() {
    fmt.Printf("Usage:%v <Listen Addr> \n", os.Args[0])
    fmt.Printf("e.g. %v 127.0.0.1:18600 \n", os.Args[0])

    listenAddr :="127.0.0.1:18600"
    if len(os.Args) >= 2 {
        listenAddr = os.Args[1]
    } else {
        fmt.Printf("Using default listen address: %v \n", listenAddr)
    }
    
	listener, err := net.Listen("tcp", listenAddr)
    handleError(err)
    defer listener.Close()
    for {
        tcpConn,err := listener.Accept()
        fmt.Printf("The client:%s has connected!\n",tcpConn.RemoteAddr().String())
        handleError(err)
        defer tcpConn.Close()
        go handleConn(tcpConn)    //起一个goroutine处理
    }
}


