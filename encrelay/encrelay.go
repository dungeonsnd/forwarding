
// 加密中继. 有点像 

package main

import (
	"bytes"
	"flag"
	"io"
	"log"
	"net"
    "time"
	"strings"
    "encoding/hex"
	"crypto/aes"
    "crypto/sha256"
	"crypto/cipher"
    "github.com/satori/go.uuid"
	"golang.org/x/crypto/pbkdf2"
)

////////////////// encryption begin //////////////////
const (
	CACRYPTO_AES_BLOCKBYTES = 16
	CACRYPTO_AES_KEYSIZE    = 32

	CACRYPTO_ENCPACKET_ROUNDS = 2

	CACRYPTO_PBKDFRounds = 4
)

var CACRYPTO_CipherPWD = []byte{'!', 't', 'T', '^', 0x02, ';', '.', 'Z', '@', '~', '(', '$', '3', 'f', '7'}
var CACRYPTO_CipherSALT = []byte{'?', 't', 'C', 'z', 0x04, 'S', '<', 'E', '@', '~', '|', '$', '2', 'f', '0'}
var CACRYPTO_CipherIV = []byte{'c', '#', 'Z', '%', 0x08, 'T', '!', 'M', '=', '+', '&', '$', '6', 'f', 'Q'}
////////////////// encryption end //////////////////


var (
	listen   = flag.String("listen", "127.0.0.1:38601", "Address to listen. Can be empty, default is 127.0.0.1:38601.")
	connect  = flag.String("connect", "127.0.0.1:38602", "Address to connect. Can be empty, default is 127.0.0.1:38602.")
    mode     = flag.String("mode", "0", "encryption type. 0(Transparent): Transmit directly. 1(Source):Encrypt accept's recved data, decrypt connect's recved data. 2(Target): Decrypt accept's recved data, encrypt connect's recved data. ")
	algo     = flag.String("algo", "aes-256-ctr", "encryption algorithm. Can be empty, default is aes-256-ctr.")
	parm     = flag.String("parm", "", "encryption parameters. Can be empty, use default.")
	pwd      = flag.String("pwd", "", "encryption password. Can be empty, use default.")
)

type ConnInfo struct {
	conn    net.Conn // tcp连接
	chid    string   // channel id
	timeout uint32   // 客户端连接的心跳超时
	anotherChid    string   // 另一方的 channel id
}

type SendQEle struct {
	chid string        // 发送者
	data *bytes.Buffer // 发送的数据
}

var cmdQ = make(chan string)           // gorouting 控制命令
var sendQ = make(chan SendQEle)        // 要发送的数据发给 Dispatcher
var createConnQ = make(chan *ConnInfo) // 连接创建
var removeConnQ = make(chan string)    // 连接关闭

func main() {
	flag.Parse()
    
//  go run EchoServer.go 7900
	log.Printf("e.g. of Mode 2 (Target):  go run encrelay.go -listen :7800 -connect 127.0.0.1:7900 -mode 2 -pwd xwxwxw \n\n")
	log.Printf("e.g. of Mode 1 (Source):  go run encrelay.go -listen :7700 -connect 127.0.0.1:7800 -mode 1 -pwd xwxwxw \n")
//  telnet 127.0.0.1 7700
    
	log.Printf("------------------------------------------------- \n\n\n")
	log.Printf("Program Started. listen:%v, connect:%v, mode:%v, algo:%v \n\n", *listen, *connect, *mode, *algo)

	go Dispatcher()
	go Accepter(*listen, *connect, *mode, *algo, *parm, *pwd)

	select {
	case cmd := <-cmdQ: // 收到控制指令
		if strings.EqualFold(cmd, "quit") {
			log.Println("quit")
			break
		}
	}
}

// 管理所有客户端连接信息； 发送数据
func Dispatcher() {
	connMap := make(map[string]*ConnInfo)
	for {
		select {
		case cmd := <-cmdQ:
			if strings.EqualFold(cmd, "quit") {
				break
			}

		case sendQEle := <-sendQ: // 从发送队列取出 chid->bytes ，找出接收者，然后发送出去。
			senderInfo, found := connMap[sendQEle.chid]
			if !found {
				continue
			}
			recverInfo, found := connMap[senderInfo.anotherChid]
			if !found {
				continue
			}
            
            nTotal :=len(sendQEle.data.Bytes())
            nWritten :=0
            for {
                n, err := recverInfo.conn.Write(sendQEle.data.Bytes())
                if err != nil {
                    log.Printf("Write data length: %v to: %v(%v) ERROR: %v \n\n", 
                        nTotal, recverInfo.chid, recverInfo.conn.RemoteAddr(), err)
                    break
                } else {
                    nWritten +=n
                    if nWritten==nTotal {
                        log.Printf("Write data length: %v to: %v(%v) \n\n", 
                            nTotal, recverInfo.chid, recverInfo.conn.RemoteAddr())
                        break
                    }
                }
            }

		case connInfo := <-createConnQ: // 接收到连接创建通知时，创建连接信息。
			connMap[connInfo.chid] = connInfo
			log.Printf("Create connection: %v(%v) [%v] \n\n", 
                connInfo.chid, connInfo.conn.RemoteAddr(), connInfo.anotherChid)

		case chid := <-removeConnQ: // 接收到连接关闭通知时，移除连接信息。
			connInfo, found := connMap[chid]
			if !found {
				continue
			}
			delete(connMap, chid)
			log.Printf("Remove connection: %v(%v) \n\n", chid, connInfo.conn.RemoteAddr())
            
            // 关闭关联的另一连接
			anotherConnInfo, found := connMap[connInfo.anotherChid]
			if !found {
				continue
			}
            anotherConnInfo.conn.Close()
            log.Printf("Close connection: %v(%v) \n\n", anotherConnInfo.chid, connInfo.conn.RemoteAddr())     
		}
	}
}

// 创建侦听; 创建 Handler
func Accepter(listen string, connect string, mode string, algo string, parm string, pwd string) {

	listener, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatal(err)
		cmdQ <- "quit"
		return
	}

	for {
		serverConn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
        defer serverConn.Close()
        
        serverChid :=genNewChid()
        log.Printf("Accept from %v(%v) \n\n", serverChid, serverConn.RemoteAddr())
        
        go clientHandler(connect, serverConn, serverChid, mode, algo, parm, pwd)
	}
}

func clientHandler(connect string, serverConn net.Conn, serverChid string, mode string, algo string, parm string, pwd string) {

    var clientConn net.Conn
    for {
        c, err := net.Dial("tcp", connect)
        if err != nil {
            log.Printf("Connect %s failed! \n", connect)
            time.Sleep(1000 * time.Millisecond)
            continue
        } else {
            clientConn =c
            break
        }
    }
    defer clientConn.Close()
    
    clientChid :=genNewChid()
    log.Printf("Connected to %v(%v) \n\n", clientChid, clientConn.RemoteAddr())
    createConnQ <- &ConnInfo{clientConn, clientChid, 60*15, serverChid}   
    
    go serverHandler(serverConn, serverChid, clientChid, mode, algo, parm, pwd)     
    ReadAndSendDataLoop(clientConn, clientChid, false, mode, pwd)
}

func serverHandler (serverConn net.Conn, serverChid string, clientChid string, mode string, algo string, parm string, pwd string) {
    createConnQ <- &ConnInfo{serverConn, serverChid, 60*15, clientChid}
    ReadAndSendDataLoop(serverConn, serverChid, true, mode, pwd)
}

func genNewChid() string {
	chid := uuid.NewV4()
    hash := sha256.New()
    hash.Write(chid[0:])
    md := hash.Sum(nil)
    mdStr := hex.EncodeToString(md[0:2])
    return mdStr
}

func ReadAndSendDataLoop(conn net.Conn, chid string, isServer bool, mode string, pwd string) {
	for {
		data := make([]byte, 1024*32)
		n, err := conn.Read(data) // 读出数据，放入 Dispatcher 的发送队列
		if err != nil {           // 关闭时给 Dispatcher 发送通知
			if err == io.EOF {
				log.Printf("Connection %v(%v) closed! \n\n", chid, conn.RemoteAddr())
			} else {
				log.Printf("Read from %v(%v) error: %v \n\n", chid, conn.RemoteAddr(), err)
			}
			removeConnQ <- chid
			break
		}
        
        if n>0 {
            recved :=data[:n]
            log.Printf("Read data length:%v from %v(%v), mode:%v, isServer:%v\n\n",
                len(recved), chid, conn.RemoteAddr(), mode, isServer)
            
            // mode
            // 0(Transparent): Transmit directly. 
            // 1(Source):Encrypt accept's recved data, decrypt connect's recved data. 
            // 2(Target): Decrypt accept's recved data, encrypt connect's recved data.     
            if mode=="0" {
                sendQ <- SendQEle{chid, bytes.NewBuffer(recved)}
                log.Printf("Read, transparent directly. Recved data:%v \n\n", string(recved))
            } else {
                if (mode=="1" && isServer) || (mode=="2" && isServer==false) {
                    output, err :=AESEncPacket(pwd, recved)
                    if err!=nil {
                        log.Printf("AESEncPacket failed, err=%v \n\n", err)
                    } else {
                        sendQ <- SendQEle{chid, bytes.NewBuffer(output)}
                        log.Printf("Read, encrypt and send. Recved data:%v \n\n", string(recved))
                    }
                } else if (mode=="1" && isServer==false) || (mode=="2" && isServer) {
                    output, err :=AESDecPacket(pwd, recved)
                    if err!=nil {
                        log.Printf("AESDecPacket failed, err=%v \n\n", err)
                    } else {
                        sendQ <- SendQEle{chid, bytes.NewBuffer(output)}
                        log.Printf("Read, decrypt and send. Recved data:%v\n\n", string(recved))
                    }
                }
            }
            
        }
	}
}


//////////////// encryption /////////////////////
func CalHash(src []byte, rounds int) []byte {
	dst := src
	for i := 0; i < rounds; i++ {
		t := sha256.Sum256(dst)
		dst = t[0:]
	}
	return dst
}

func CalHashHex(src []byte, rounds int) string {
	h := CalHash(src, rounds)
	return hex.EncodeToString(h)
}

func genPacketPWD(pwd string) string {
	return CalHashHex([]byte(pwd), CACRYPTO_ENCPACKET_ROUNDS)
}

func genPacketSALT() []byte {
	return CalHash(CACRYPTO_CipherSALT, CACRYPTO_ENCPACKET_ROUNDS)
}

func genPacketIV() []byte {
	h := CalHash(CACRYPTO_CipherIV, CACRYPTO_ENCPACKET_ROUNDS)
	return h[0:CACRYPTO_AES_BLOCKBYTES]
}

func deriveKey(pwd string) []byte {
	passwd := genPacketPWD(pwd)
	salt := genPacketSALT()
	resultLen := CACRYPTO_AES_KEYSIZE
	key := pbkdf2.Key([]byte(passwd), salt, CACRYPTO_PBKDFRounds, resultLen, sha256.New)
	return key
}

func AESEncPacket(pwd string, input []byte) (output []byte, err error) {
	key := deriveKey(pwd)
	iv := genPacketIV()
	block, err1 := aes.NewCipher(key)
	if err1 != nil {
		err = err1
		return
	}
	aes := cipher.NewCTR(block, iv)
	output = make([]byte, len(input))
	aes.XORKeyStream(output, input)
	return
}

func AESDecPacket(pwd string, input []byte) (output []byte, err error) {
	key := deriveKey(pwd)
	iv := genPacketIV()
	block, err1 := aes.NewCipher(key)
	if err1 != nil {
		err = err1
		return
	}
	aes := cipher.NewCTR(block, iv)
	output = make([]byte, len(input))
	aes.XORKeyStream(output, input)
	return
}

