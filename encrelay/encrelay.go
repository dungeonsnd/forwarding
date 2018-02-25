//
// 加密代理程序
// version: 0.1
// 功能: 在两台计算机之间建立加密的网络连接.
//
// 
// 使用举例:
// 通过 encrelay ，使 realvnc/vnc-viewer 从远端连接到家里 realvnc/vnc-server，所有数据经过 encrelay 进行加密传输.
//
// 步骤 1) 家里 vnc-server 启动端口为 5900.   在本机启动 encrelay 服务端 (mode=2),
//         go run encrelay.go -listen :9001 -connect 127.0.0.1:5900 -mode 2 -pwd x123y456z789
// 步骤 2) 在远端机启动 encrelay 客户端(mode=1),
//         go run encrelay.go -listen :7700 -connect [home ip address]:9001 -mode 1 -pwd x123y456z789
// 步骤 3) 在远端启动 realvnc/vnc-viewer，连接 127.0.0.1:7700
//

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/hex"
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/satori/go.uuid"
	"golang.org/x/crypto/pbkdf2"
)

////////////////// encryption begin //////////////////
const (
	caCryptoAesBlockBytes = 16
	caCryptoAesKeySize    = 32

	caCryptoEncPacketRounds = 2
	caCryptoPbkdfRounds     = 4
)

var caCryptoCipherSalt = []byte{'?', 't', 'C', 'z', 0x04, 'S', '<', 'E', '@', '~', '|', '$', '2', 'f', '0'}
var caCryptoCipherIv = []byte{'c', '#', 'Z', '%', 0x08, 'T', '!', 'M', '=', '+', '&', '$', '6', 'f', 'Q'}

var passwd ="Ac1M..1z2z3z"

var (
	listen  = flag.String("listen", "127.0.0.1:38601", "Address to listen. Can be empty, default is 127.0.0.1:38601.")
	connect = flag.String("connect", "127.0.0.1:38602", "Address to connect. Can be empty, default is 127.0.0.1:38602.")
	mode    = flag.String("mode", "0", "encryption type. 0(Transparent): Transmit directly. 1(Source):Encrypt accept's recved data, decrypt connect's recved data. 2(Target): Decrypt accept's recved data, encrypt connect's recved data. ")
	pwd     = flag.String("pwd", "123123", "encryption password. Can be empty, use default.")
)

type connInfo struct {
	conn        net.Conn // tcp连接
	chid        string   // channel id
	timeout     uint32   // 客户端连接的心跳超时
	anotherChid string   // 另一方的 channel id
}

type sendQEle struct {
	chid string        // 发送者
	data *bytes.Buffer // 发送的数据
}

var cmdQ = make(chan string)           // gorouting 控制命令
var sendQ = make(chan sendQEle)        // 要发送的数据发给 Dispatcher
var createConnQ = make(chan *connInfo) // 连接创建
var removeConnQ = make(chan string)    // 连接关闭

func main() {
	flag.Parse()

	//  go run EchoServer.go 5900
	log.Printf("e.g. of Mode 2 (Target):  go run encrelay.go -listen :9001 -connect 127.0.0.1:5900 -mode 2 -pwd x123y456z789 \n\n")
	log.Printf("e.g. of Mode 1 (Source):  go run encrelay.go -listen :7700 -connect [home ip address]:9001 -mode 1 -pwd x123y456z789 \n")
	//  telnet 127.0.0.1 7700

	// log.Printf("------------------------------------------------- \n\n\n")
	log.Printf("Program Started. listen:%v, connect:%v, mode:%v \n\n", *listen, *connect, *mode)
    passwd =*pwd

	go Dispatcher()
	go Accepter(*listen, *connect, *mode)

	select {
	case cmd := <-cmdQ: // 收到控制指令
		if strings.EqualFold(cmd, "quit") {
			log.Println("quit")
			break
		}
	}
}

// Dispatcher 管理所有客户端连接信息； 发送数据
func Dispatcher() {
	connMap := make(map[string]*connInfo)
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

			nTotal := len(sendQEle.data.Bytes())
			nWritten := 0
			for {
				n, err := recverInfo.conn.Write(sendQEle.data.Bytes())
				if err != nil {
					log.Printf("Write data length: %v to: %v(%v) ERROR: %v \n\n",
						nTotal, recverInfo.chid, recverInfo.conn.RemoteAddr(), err)
					break
				} else {
					nWritten += n
					if nWritten == nTotal {
						log.Printf("<<== Write data length: %v to: %v(%v) \n\n",
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

// Accepter : 创建侦听; 创建 Handler
func Accepter(listen string, connect string, mode string) {

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

		serverChid := genNewChid()
		log.Printf("Accept from %v(%v) \n\n", serverChid, serverConn.RemoteAddr())

		go clientHandler(connect, serverConn, serverChid, mode)
	}
}

func clientHandler(connect string, serverConn net.Conn, serverChid string, mode string) {

	var clientConn net.Conn
	for {
		c, err := net.Dial("tcp", connect)
		if err != nil {
			log.Printf("Connect %s failed! \n", connect)
			time.Sleep(1000 * time.Millisecond)
			continue
		} else {
			clientConn = c
			break
		}
	}
	defer clientConn.Close()

	clientChid := genNewChid()
	log.Printf("Connected to %v(%v) \n\n", clientChid, clientConn.RemoteAddr())
	createConnQ <- &connInfo{clientConn, clientChid, 60 * 15, serverChid}

	go serverHandler(serverConn, serverChid, clientChid, mode)
	ReadAndSendDataLoop(clientConn, clientChid, false, mode)
}

func serverHandler(serverConn net.Conn, serverChid string, clientChid string, mode string) {
	createConnQ <- &connInfo{serverConn, serverChid, 60 * 15, clientChid}
	ReadAndSendDataLoop(serverConn, serverChid, true, mode)
}

func genNewChid() string {
	chid := uuid.NewV4()
	hash := sha256.New()
	hash.Write(chid[0:])
	md := hash.Sum(nil)
	mdStr := hex.EncodeToString(md[0:3])
	return mdStr
}

func readAndUnpack(conn net.Conn, chid string, data []byte) (int, error){
    // Read header
    log.Printf("Before ReadData, %v(%v) \n\n", chid, conn.RemoteAddr())
    header, err := ReadData(conn, 8)
    if err != nil {            // 关闭时给 Dispatcher 发送通知
        if err == io.EOF || err == io.ErrUnexpectedEOF{
            log.Printf("Connection %v(%v) closed! error: %v \n\n", chid, conn.RemoteAddr(), err)
        } else {
            log.Printf("Read from %v(%v) error: %v \n\n", chid, conn.RemoteAddr(), err)
        }
        removeConnQ <- chid
        return 0, err
    }
    
    // Calculate bodyLen
    bodyLen, err :=BytesToInt32(header[0:4])
    if err != nil || bodyLen<0 || bodyLen>1024*1024*64 {
        log.Printf("BytesToInt32 error. bodyLen=%v. Close connection %v(%v)! error: %v \n\n", 
                    bodyLen, chid, conn.RemoteAddr(), err)
        conn.Close() // 主动关闭.
        removeConnQ <- chid
        return 0, err
    }
    
    // Read body
    if len(data)<int(bodyLen) {
        data = make([]byte, bodyLen)
    }    
    log.Printf("Before ReadData, bodyLen=%v, %v(%v) \n\n", bodyLen, chid, conn.RemoteAddr())
    err = ReadData2(conn, data, (int)(bodyLen))
    if err != nil {
        if err == io.EOF {
            log.Printf("Connection %v(%v) closed! \n\n", chid, conn.RemoteAddr())
        } else {
            log.Printf("Read from %v(%v) error: %v \n\n", chid, conn.RemoteAddr(), err)
        }
        removeConnQ <- chid
        return 0, err
    }
    return (int)(bodyLen), nil
}

func readFullData(conn net.Conn, chid string, data []byte) (int, error) {
    log.Printf("Before readFullData, %v(%v) \n\n", chid, conn.RemoteAddr())
    bodyLen, err := conn.Read(data) // 读出数据，放入 Dispatcher 的发送队列
    if err != nil {           // 关闭时给 Dispatcher 发送通知
        if err == io.EOF {
            log.Printf("Connection %v(%v) closed! \n\n", chid, conn.RemoteAddr())
        } else {
            log.Printf("Read from %v(%v) error: %v \n\n", chid, conn.RemoteAddr(), err)
        }
        removeConnQ <- chid
        return 0, err
    }
    return (int)(bodyLen), nil
}

func packData(output []byte) (* bytes.Buffer, error){
    bodyLen, err :=Int32ToBytes((int32)(len(output)))
    if err != nil {
        log.Printf("Int32ToBytes failed, err=%v \n\n", err)
        return nil, err
    } else {
        
        ext := make([]byte, 4)
        b := bytes.NewBuffer(bodyLen)
        b.Write(ext)
        b.Write(output)
        
        return b, nil
    }        
}

func ReadAndSendDataLoop(conn net.Conn, chid string, isServer bool, mode string) {
    needUnpack :=(mode == "1" && isServer == false) || (mode == "2" && isServer)
    
	data := make([]byte, 1024*32)
	for {        
        bodyLen :=0
        var err error
        if needUnpack {
            bodyLen, err =readAndUnpack(conn, chid, data)
        } else {
            bodyLen, err =readFullData(conn, chid, data)
        }
        if err!=nil {
            log.Printf("read failed, err=%v \n\n", err)
            break
        }
        if bodyLen <= 0 {
            log.Printf("bodyLen=%v \n\n", bodyLen)
            continue
        }

        recved := data[:bodyLen]
        log.Printf("==>> Read data length:%v from %v(%v), mode:%v, isServer:%v\n\n",
            len(recved), chid, conn.RemoteAddr(), mode, isServer)

        // mode
        // 0(Transparent): Transmit directly.
        // 1(Source):Encrypt accept's recved data, decrypt connect's recved data.
        // 2(Target): Decrypt accept's recved data, encrypt connect's recved data.
        if mode == "0" {
            sendQ <- sendQEle{chid, bytes.NewBuffer(recved)}
            // log.Printf("Read, transparent directly. Recved data:%v \n\n", string(recved))
            
        } else {
            if (mode == "1" && isServer) || (mode == "2" && isServer == false) {
                output, err := AESEncPacket(passwd, recved)
                if err != nil {
                    log.Printf("AESEncPacket failed, err=%v \n\n", err)
                    conn.Close()
                    removeConnQ <- chid
                    break
                }
                
                // package
                b, err :=packData(output)
                if err != nil {
                    log.Printf("packData failed, err=%v \n\n", err)
                    conn.Close()
                    removeConnQ <- chid
                    break
                }
                sendQ <- sendQEle{chid, b}
                //log.Printf("Read, encrypt , package, and send. Recved data:%v \n\n", string(recved))
                
            } else if (mode == "1" && isServer == false) || (mode == "2" && isServer) {
                output, err := AESDecPacket(passwd, recved)
                if err != nil {
                    log.Printf("AESDecPacket failed, err=%v \n\n", err)
                    conn.Close()
                    removeConnQ <- chid
                    break
                }
                
                sendQ <- sendQEle{chid, bytes.NewBuffer(output)}
                //log.Printf("Read, decrypt and send. Recved data:%v\n\n", string(recved))
            }
        }
	}
}


///////////////////////// i/o /////////////////////////
func ReadData(conn net.Conn, total int) (buf []byte, err error) {
	buf = make([]byte, total)
	if _, err1 := io.ReadFull(conn, buf); err1 != nil {
		err = err1
	}
	return
}

func ReadData2(conn net.Conn, buf []byte, total int) (err error) {
	if _, err1 := io.ReadFull(conn, buf[:total]); err1 != nil {
		err = err1
	}
	return
}

func WriteData(conn net.Conn, buf []byte) (err error) {
	total := len(buf)
	for nw := 0; nw < total; {
		n, err1 := conn.Write(buf[nw:])
		if err1 != nil {
			err = err1
			break
		}
		nw += n
	}
	return
}


func Int32ToBytes(x int32) (buf []byte, err error) {
	b_buf := new(bytes.Buffer)
	err = binary.Write(b_buf, binary.BigEndian, x)
	buf = b_buf.Bytes()
	return
}

func BytesToInt32(buf []byte) (x int32, err error) {
	b_buf := bytes.NewBuffer(buf)
	err = binary.Read(b_buf, binary.BigEndian, &x)
	return
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
	return CalHashHex([]byte(pwd), caCryptoEncPacketRounds)
}

func genPacketSALT() []byte {
	return CalHash(caCryptoCipherSalt, caCryptoEncPacketRounds)
}

func genPacketIV() []byte {
	h := CalHash(caCryptoCipherIv, caCryptoEncPacketRounds)
	return h[0:caCryptoAesBlockBytes]
}

func deriveKey(pwd string) []byte {
	passwd := genPacketPWD(pwd)
	salt := genPacketSALT()
	resultLen := caCryptoAesKeySize
	key := pbkdf2.Key([]byte(passwd), salt, caCryptoPbkdfRounds, resultLen, sha256.New)
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
