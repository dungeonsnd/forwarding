package main

import (
    "net"
    "fmt"
    "os"
    "time"
    "bufio"
)

const BUFF_SIZE = 1024
var input = make([]byte,BUFF_SIZE)

func handleError(err error) {
    if err == nil {
        return
    }
    fmt.Printf("error:%s\n",err.Error())
}

func main() {
    fmt.Printf("Usage:%v <Server Addr> \n", os.Args[0])
    fmt.Printf("e.g. %v 127.0.0.1:18600 \n", os.Args[0])

    serverAddr :="127.0.0.1:18600"
    if len(os.Args) >= 2 {
        serverAddr = os.Args[1]
    } else {
        fmt.Printf("Using default server address: %v \n", serverAddr)
    }
    
    tcpConn,err := net.Dial("tcp",serverAddr)
    handleError(err)
    reader :=  bufio.NewReader(os.Stdin)
    
    var continued = true
    var inputStr string
    for continued {
        n,err := reader.Read(input)
        handleError(err)
        if n > 0 {
            k,_ := tcpConn.Write(input[:n])
            if k > 0 {
                inputStr = string(input[:k])
                if inputStr == "exit\n" {  //在比对时由于有个回车符，所以加上\n
                    continued = false        //也可以将inputStr = TrimRight(inputStr,"\n")
                }
                
                n,err := tcpConn.Read(input)
                if err != nil {
                    fmt.Printf("The RemoteAddr:%s is closed! err:%v \n\n",
                        tcpConn.RemoteAddr().String(), err)
                    break
                }
                handleError(err)
                if n > 0 {
                    fmt.Printf("<<==[%v %v]: %v \n\n", 
                        time.Now().Format("2006-01-02 15:04:05"), 
                        tcpConn.RemoteAddr().String(),
                        string(input[:n]))
                }
            }
        }
    }
}
  
  