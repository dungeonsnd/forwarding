//
// 支持双向认证的加密代理程序 encproxy
// version: 0.1
//
// 功能: 在两台计算机之间建立双向认证的加密网络连接.
//
//
// 这个程序是模仿 openpgp 和 tls 协议来写的一个简单的加密代理，演示了在不安全信道上建立安全通信通道的常见方法。
// 本来只是想学习研究和自用的，但是发现可用性和性能还可以。于是分享出来供交流和把玩。 但是本程序的安全性和稳定性没有经过认真设计和验证，所以肯定没有 openpgp和tls 安全性高，请不要把本程序用于安全性要求较高的环境。
//  由于历史原因，代码使用了我的另一项目中的现成结构(可以从任一连接发送数据到任意连接上)，所以可能复杂性高一些。其实 proxy-client(或proxy-server)端一个 gorutine 收数据->解密->发给调用者，一个 gorutine 收数据->加密->发给proxy-server， 这样的结构对于本程序来说，可能更简单明了一些。
//
// 本程序功能基本可用了，但是还有几个特性没有完成:
// 1) 指纹 fingerprint 需要存储在磁盘中，以便下次连接请求到来时直接验证。避免每次提醒用户未知指纹.
// 2) 需要完成 nickname 的加密和交换, 并且 nickname 中需要放入加密的hash(nickname)。 以便握手双方可以验证协商出来的会话密钥是否一致。
// 3) 需要一个 UI 前端，对用户更友好一些。所以本程序也要增加 RPC 功能来与UI程序交互。
// 4) 增加其它语言平台的支持，如 iOS/OC , java/Android 平台。
// 
//
//
//
// 使用举例:
// 通过 encproxy ，使 realvnc/vnc-viewer 从远端连接到家里 realvnc/vnc-server，所有数据经过 encproxy 进行加密传输.
//
// 步骤 1) 家里 vnc-server 启动端口为 5900.   在本机启动 encproxy 服务端 (mode=2), agree=1 表示自动同意未知指纹.
//         go run encproxy.go -listen :9001 -connect 127.0.0.1:5900 -mode 2 -agree 1
// 步骤 2) 在远端机启动 encproxy 客户端(mode=1), agree=0 表示未知指纹时需要用户手动同意.
//         go run encproxy.go -listen :7700 -connect [home ip address]:9001 -mode 1 -agree 0
// 步骤 3) 在远端启动 realvnc/vnc-viewer，连接 127.0.0.1:7700
//
//  
//


// Simple crypto protocol
// version:0x00
//
// Handshaking:
// [Body length(4 Bytes)]
//
// After handshake:
// [Body length(4 Bytes)] + [Encrypted data]
//
// Using crypto algorithm:
// RSA-2048, rsa.EncryptOAEP, rsa.SignPSS, ripemd160, SHA-256, AES-256-CTR, pbkdf2.Key
//


package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
    "crypto/rsa"
    "crypto/x509"
	"encoding/hex"
    "encoding/base32"
	"encoding/binary"
	"flag"
	"io"
	"log"
    "os"
    "fmt"
	"net"
	"strings"
	"time"
    "crypto"
    "runtime"
	"strconv"
    "os/signal"
    "syscall"
    "crypto/rand"

	"github.com/satori/go.uuid"
	"golang.org/x/crypto/pbkdf2"
    "golang.org/x/crypto/ripemd160"
)

////////////////// encryption begin //////////////////
const (
	caCryptoAesBlockBytes = 16
	caCryptoAesKeySize    = 32

	caCryptoEncPacketRounds = 2
	caCryptoPbkdfRounds     = 4
)

const (
    caProtocolVersion       = 0

    handshakeStepWillSendMyVersion     = 1 // 将要发送我的版本号
    handshakeStepWaitPeerVersion     = 2 // 等待对方版本号
    handshakeStepWaitPeerPublicKey     = 3 // 等待对方公钥 (fingerprint=ripemd160(dhash(public key)))
    handshakeStepWaitPeerRandomDataToSign  = 4 // 等待对方随机数据来签名
    handshakeStepWaitPeerSignedData  = 5 // 等待对方签名的数据
    handshakeStepWaitPeerRandomSeed        = 6 // 等待对方随机种子(Encrypted by peer public key. session key=dhash(smaller  seed+bigger seed+smaller sign data+bigger sign data), iv=dhash(bigger seed+smaller seed))
    handshakeStepWaitPeerNickname      = 7// 等待对方昵称
    handshakeStepSucessfully      = 8 // 握手成功
    handshakeStepFailed      = 9 // 握手失败
)

type handshakeContext struct {
    handshakeStep int
    myPrivateKey * rsa.PrivateKey
    peerPublickKey * rsa.PublicKey
    myRandomDataToSign []byte
    peerRandomDataToSign []byte
    myRandomSeed []byte
    peerRandomSeed []byte
    sessionKey []byte
}


var caCryptoCipherSalt = []byte{'?', 't', 'C', 'z', 0x04, 'S', '<', 'E', '@', '~', '|', '$', '2', 'f', '0'}
var caCryptoCipherIv = []byte{'c', '#', 'Z', '%', 0x08, 'T', '!', 'M', '=', '+', '&', '$', '6', 'f', 'Q'}

    
var maxNetowrkConnections = 1000 // 允许的最多连接数量, 包含主动连接和接受的连接.

var (
	listen  = flag.String("listen", "127.0.0.1:38601", "Address to listen. Can be empty, default is 127.0.0.1:38601.")
	connect = flag.String("connect", "127.0.0.1:38602", "Address to connect. Can be empty, default is 127.0.0.1:38602.")
	mode    = flag.String("mode", "0", "encryption type. 0(Transparent): Transmit directly. 1(Source):Encrypt accept's recved data, decrypt connect's recved data. 2(Target): Decrypt accept's recved data, encrypt connect's recved data. ")
    connectorCount = flag.Int("count", 500, "Max count of connectors.")
    autoAgreeUnknowFingerprint = flag.Int("agree", 1, "Auto agree the unknow fingerprint. 1: Auto agree, 0: Prompt user.")
)

type connInfo struct {
	conn             net.Conn // tcp连接
	chid             string   // channel id
	timeout          uint32   // 客户端连接的心跳超时. 暂时没用到.
	anotherChid      string   // 另一方的 channel id
}

type sendQEle struct {
	chid string        // 发送者
	data *bytes.Buffer // 发送的数据
}

var cmdQ = make(chan string)           // gorouting 控制命令
var sendQ = make(chan sendQEle)        // 要发送的非握手数据发给 Dispatcher
var sendQForHandshaking = make(chan sendQEle)        // 要发送的握手数据发给 Dispatcher
var createConnQ = make(chan *connInfo) // 连接创建
var removeConnQ = make(chan string)    // 连接关闭
var netowrkConnectionsCountUpdatedQ = make(chan int, 1024)    // 网络连接数量变化通知

func main() {
	flag.Parse()

	//  go run EchoServer.go 5900
	log.Printf("e.g. of Mode 2 (Target):  go run encproxy.go -listen :9001 -connect 127.0.0.1:5900 -mode 2 -count 100 -agree 1 \n\n")
	log.Printf("e.g. of Mode 1 (Source):  go run encproxy.go -listen :7700 -connect [home ip address]:9001 -mode 1 -count 100 -agree 0 \n")
	//  telnet 127.0.0.1 7700

	log.Printf("------------------------------------------------- \n\n\n")
	log.Printf("Program Started. listen:%v, connect:%v, mode:%v, connectorCount:%v \n\n", *listen, *connect, *mode, *connectorCount)
    showMyFingerPrint()

    if *connectorCount>0 {
        maxNetowrkConnections = *connectorCount * 2
    }

    agree :=true
    if *autoAgreeUnknowFingerprint==0 {
        agree =false
    }
    
	go Dispatcher()
	go Accepter(*listen, *connect, *mode, agree)

    // 截获退出信号
    c := make(chan os.Signal, 1)
    signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

    for {
        select {
            case sig := <- c :
                switch sig {
                case syscall.SIGINT, syscall.SIGTERM, os.Interrupt: // 获取到退出信号
                    log.Printf("\nRcved exit signal: %v \n\n", sig)
                    return
                }

            case cmd := <-cmdQ: // 收到控制指令
                if strings.EqualFold(cmd, "quit") {
                    log.Printf("\nRcved quit command. \n\n")
                    return
                }
        }
    }
}

func showMyFingerPrint() {
    // 生成或读取我的公钥
    ret, _, myPublicKeyBuf :=getMyPublicKey()
    if !ret {
        return
    }
        
    fingerprint := calFingerPrint(myPublicKeyBuf)
    log.Printf("\n********************************************************\nMy fingerprint is:\n%v\n********************************************************\n", fingerprint)
}

// Dispatcher 管理所有客户端连接信息； 发送数据
func Dispatcher() {
	connMap := make(map[string]*connInfo)
	for {
		select {

		case sendQEle := <-sendQ: // 从发送队列取出 chid->bytes ，找出关联的接收者连接，然后发送出去。
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
//						log.Printf("<<== Write data length: %v to: %v(%v) \n\n",
//							nTotal, recverInfo.chid, recverInfo.conn.RemoteAddr())
						break
					}
				}
			}

		case sendQEle := <-sendQForHandshaking: // 从握手发送队列取出 chid->bytes ，直接向本连接发送出去。
			senderInfo, found := connMap[sendQEle.chid]
			if !found {
				continue
			}

			nTotal := len(sendQEle.data.Bytes())
			nWritten := 0
			for {
				n, err := senderInfo.conn.Write(sendQEle.data.Bytes())
				if err != nil {
					log.Printf("Write data length: %v to: %v(%v) ERROR: %v \n\n",
						nTotal, senderInfo.chid, senderInfo.conn.RemoteAddr(), err)
					break
				} else {
					nWritten += n
					if nWritten == nTotal {
						log.Printf("<<== Write data length: %v to: %v(%v) \n\n",
							nTotal, senderInfo.chid, senderInfo.conn.RemoteAddr())
						break
					}
				}
			}

		case connInfo := <-createConnQ: // 接收到连接创建通知时，创建连接信息。                    
			connMap[connInfo.chid] = connInfo
			log.Printf("Create connection: %v(%v) [%v] \n\n",
				connInfo.chid, connInfo.conn.RemoteAddr(), connInfo.anotherChid)

            netowrkConnectionsCountUpdatedQ <- len(connMap)

		case chid := <-removeConnQ: // 接收到连接关闭通知时，移除连接信息。
			connInfo, found := connMap[chid]
			if !found {
				continue
			}
            log.Printf("Close connection: %v(%v) \n\n", chid, connInfo.conn.RemoteAddr())
            connInfo.conn.Close()
            delete(connMap, chid)

			// 关闭关联的另一连接
			anotherConnInfo, found := connMap[connInfo.anotherChid]
			if !found {
				continue
			}
            log.Printf("Close anotherConnInfo: %v(%v) \n\n", anotherConnInfo.chid, anotherConnInfo.conn.RemoteAddr())
			anotherConnInfo.conn.Close()
            delete(connMap, anotherConnInfo.chid)

            netowrkConnectionsCountUpdatedQ <- len(connMap)
		}
	}
}

// Accepter : 创建侦听; 创建 Handler
func Accepter(listen string, connect string, mode string, autoAgreeUnknowFingerprint bool) {

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
        SetReadTimeout(serverConn, 60*60)

        // 接收网络连接数更新通知, 或者一定时间收不到通知做超时处理
        currentNetowrkConnectionsCount := 0
        var timer *time.Timer
        stop := false
        for {
            select {
            case <- func() <-chan time.Time {
                if timer ==nil {
                    timer =time.NewTimer(2*time.Second)
                } else {
                    timer.Reset(2*time.Second)
                }
                return timer.C
            }():
                log.Printf("netowrkConnectionsCountUpdatedQ timer, connectionsCount{%v}, max:%v \n\n",
                    currentNetowrkConnectionsCount, maxNetowrkConnections)
                stop = true
                break

            case currentNetowrkConnectionsCount = <- netowrkConnectionsCountUpdatedQ:
                log.Printf("chan of netowrkConnectionsCountUpdatedQ recved, connectionsCount{%v}, maxCount:%v \n\n",
                    currentNetowrkConnectionsCount, maxNetowrkConnections)
            }

            if stop {
                break
            }
        }

        // 如果连接数较大时关闭当前连接,继续下一轮侦听.
        if currentNetowrkConnectionsCount > maxNetowrkConnections {
            log.Printf("currentNetowrkConnectionsCount{%v}>{%v} Sleep, close this accepted connection, and continue accept. \n\n",
                currentNetowrkConnectionsCount, maxNetowrkConnections)
            time.Sleep(5000 * time.Millisecond)

        } else {
            serverChid := genNewChid()
            clientChid := genNewChid()
            log.Printf("Accept from %v(%v) \n\n", serverChid, serverConn.RemoteAddr())

            var signalHandshakeOverQ = make(chan string, 16)    // 握手结果信号, len(str)>1时表示pwd, 否则表示握手失败.
            var signalMode2ToStartConnectQ = make(chan bool, 16)    // 握手即将结束信号

            go serverHandler(serverConn, serverChid, clientChid, mode, autoAgreeUnknowFingerprint, signalHandshakeOverQ, signalMode2ToStartConnectQ)
            go clientHandler(connect, serverConn, serverChid, clientChid, mode, autoAgreeUnknowFingerprint, signalHandshakeOverQ, signalMode2ToStartConnectQ)
        }
        
	}
}

func clientHandler(connect string, serverConn net.Conn, serverChid string, clientChid string, mode string, autoAgreeUnknowFingerprint bool, signalHandshakeOverQ chan string, signalMode2ToStartConnectQ chan bool) {
    if mode=="2" {
        hsResult := <-signalMode2ToStartConnectQ
        if false==hsResult {
            log.Printf("Recved handshake failed signal in clientHandler. clientChid:%v \n\n", clientChid)
            return
        }
    }

	var clientConn net.Conn
	for {
		c, err := net.Dial("tcp", connect)
		if err != nil {
			log.Printf("Connect %s failed! \n", connect)
			time.Sleep(3000 * time.Millisecond)
			continue
		} else {
			clientConn = c
			break
		}
	}
	defer clientConn.Close()
    SetReadTimeout(serverConn, 60*60)

	log.Printf("Connected to %v(%v) \n\n", clientChid, clientConn.RemoteAddr())
	createConnQ <- &connInfo{clientConn, clientChid, 60 * 15, serverChid}
    
	readAndSendDataLoop(clientConn, clientChid, false, mode, autoAgreeUnknowFingerprint, signalHandshakeOverQ, signalMode2ToStartConnectQ)
}

func serverHandler(serverConn net.Conn, serverChid string, clientChid string, mode string, autoAgreeUnknowFingerprint bool, signalHandshakeOverQ chan string, signalMode2ToStartConnectQ chan bool) {
	createConnQ <- &connInfo{serverConn, serverChid, 60 * 15, clientChid}

	readAndSendDataLoop(serverConn, serverChid, true, mode, autoAgreeUnknowFingerprint, signalHandshakeOverQ, signalMode2ToStartConnectQ)
}

func genNewChid() string {
	chid := uuid.NewV4()
	hash := sha256.New()
	hash.Write(chid[0:])
	md := hash.Sum(nil)
	mdStr := hex.EncodeToString(md[0:3])
	return mdStr
}

func readAndUnpackOnEncryptedSession(conn net.Conn, chid string, data []byte) (int, error){
    // Read header
//    log.Printf("Before ReadData, %v(%v) \n\n", chid, conn.RemoteAddr())
    header, err := ReadData(conn, 4)
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
        removeConnQ <- chid
        return 0, err
    }
    
    // Read body
    if len(data)<int(bodyLen) {
        data = make([]byte, bodyLen)
    }    
//    log.Printf("Before ReadData, bodyLen=%v, %v(%v) \n\n", bodyLen, chid, conn.RemoteAddr())
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

func readFullDataOnUnencryptedSession(conn net.Conn, chid string, data []byte) (int, error) {
//    log.Printf("Before readFullData, %v(%v) \n\n", chid, conn.RemoteAddr())
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
    t :=0
    if output!=nil {
        t =len(output)
    }
    bodyLen, err :=Int32ToBytes(int32(t))
    if err != nil {
        log.Printf("Int32ToBytes bodyLen failed, err=%v \n\n", err)
        return nil, err
    } else {
        b := bytes.NewBuffer(bodyLen)
        b.Write(output)
        return b, nil
    }        
}

func packAndSendHandshakeData(chid string, data []byte) (bool) {
//    log.Printf("packAndSendHandshakeData, chid=%v, data=%v \n", chid, data)
    b, err :=packData(data)
    if err != nil {
        log.Printf("packData failed, err=%v \n\n", err)
        return false
    } else {
        sendQForHandshaking <- sendQEle{chid, b}
        return true
    }
}


func getMyVersion() (bool, []byte){      
    version, err :=Int32ToBytes(int32(caProtocolVersion))
    if err != nil {
        log.Printf("Int32ToBytes version failed, err=%v \n\n", err)
        return false, nil
    } else {
        return true, version
    }
}

func verifingPeerVersion(recved []byte) (bool){  
    peerVersion, err :=BytesToInt32(recved[0:4])
    if err != nil || peerVersion<0 {
        log.Printf("BytesToInt32 error. peerVersion=%v, error: %v \n\n", peerVersion, err)
        return false
    }
    return int32(caProtocolVersion)==peerVersion
}


func getMyPublicKey() (bool, * rsa.PrivateKey, []byte) {    
    priFileName := "mypri.key"
    mypriBuf, err := contentOfFile(priFileName)
    var myPrivateKey * rsa.PrivateKey
    if err != nil {
        rng := rand.Reader
        bits :=2048
        myPrivateKey, err = rsa.GenerateKey(rng, bits)
        if err != nil {
            log.Printf("GenerateKey failed, err=%v \n\n", err)
            return false, myPrivateKey, nil
        }
        
        mypriBuf := x509.MarshalPKCS1PrivateKey(myPrivateKey)
        err =writeToFile(priFileName, mypriBuf)
        if err != nil {
            return false, myPrivateKey, nil
        }
        
    } else {    
        myPrivateKey, err = x509.ParsePKCS1PrivateKey(mypriBuf)
        if err != nil {
            log.Printf("ParsePKCS1PrivateKey failed, err=%v \n\n", err)
            return false, myPrivateKey, nil
        }
    }
        
    myPublicKey := (myPrivateKey.Public()).(*rsa.PublicKey)
    mypubBuf := x509.MarshalPKCS1PublicKey(myPublicKey)
    if err != nil {
        return false, myPrivateKey, nil
    }        
    return true, myPrivateKey, mypubBuf
}

func dhash(buf []byte) ([]byte) {
    return CalHash(buf, 2)
}

func calFingerPrint(peerPubBuf []byte) (string) {
    //return dhash(peerPubBuf)
    hexs := hex.EncodeToString(CalRipemd160(dhash(peerPubBuf)))

    rt := []string{}
    for i, _ := range hexs {
        if i%2==0 && i<len(hexs)-1 {
            rt = append(rt, (string)(hexs[i:i+2]))            
        }
    }    
    return strings.Join(rt, " ")
}

func checkPeerPublicKey(peerPubBuf []byte, autoAgreeUnknowFingerprint bool) (bool, * rsa.PublicKey) {
    fingerPrint := calFingerPrint(peerPubBuf)
    whitelistFileName := "whitelist.txt"
    whiteFingerListByte, err := contentOfFile(whitelistFileName) 
        
    isInWhiteList :=false
    if err == nil {
        s := string(whiteFingerListByte[:])
        whiteFingerListArr := strings.Split(s, "\n")

        fmt.Printf("\nWhite Fingers :\n")
        for _, w := range whiteFingerListArr {
            w1 :=strings.TrimSpace(w)
            fmt.Printf("%v\n", w1)
        }
        fmt.Printf("\n")
        
        for _, w := range whiteFingerListArr {
            w1 :=strings.TrimSpace(w)
        
            if (strings.EqualFold(strings.ToLower(w1), strings.ToLower(fingerPrint))) {
                isInWhiteList =true
                fmt.Printf("########## Whitelist Fignerprint, Accept. 指纹在白名单中,允许连接.:\n%v\n\n", fingerPrint)
            }
        }
    }

    if false==isInWhiteList {
    
        if autoAgreeUnknowFingerprint {
            fmt.Printf("########## Unkown Fignerprint. 未知指纹:\n%v\n******** Already Config Auto-Accepted. 已配置自动允许连接\n\n\n", fingerPrint)
        } else {
            fmt.Printf("########## Unkown Fignerprint. 未知指纹:\n%v\n******** Accept or Not? 是否允许连接? Y/N ? \n", fingerPrint)
            yesorno :=""
            fmt.Scanln(&yesorno)
            if yesorno!="Y" && yesorno!="y" {
                return false, nil
            }
        }
        fmt.Printf("\n\n\n")

        err = appendToFile(whitelistFileName, []byte(fingerPrint))  // TODO: json format.
        if err != nil {
            return false, nil
        }
    }
    
    peerPublicKey, err := x509.ParsePKCS1PublicKey(peerPubBuf)
    if err != nil {
        return false, nil
    }        
    return true, peerPublicKey
}

func genRandomDataToSign() ([]byte){      
    randomDataToSign :=RandByte()
    return randomDataToSign
}

func signPeerRandomData(peerRandomData []byte, myPrivateKey * rsa.PrivateKey) (bool, []byte){
    rng := rand.Reader
    hashed := sha256.Sum256(peerRandomData)
    signedData, err := rsa.SignPSS(rng, myPrivateKey, crypto.SHA256, hashed[:], nil)
    if err != nil {
        log.Printf("SignPSS failed, err=%v \n", err)
        return false, nil
    }
    return true, signedData
}

func verifingPeerSignedData(recved []byte, peerPublickKey * rsa.PublicKey, randomDataToSign []byte) (bool){ 
    hashed := sha256.Sum256(randomDataToSign)
    err := rsa.VerifyPSS(peerPublickKey, crypto.SHA256, hashed[:], recved, nil)
    if err != nil {
        log.Printf("VerifyPSS failed, err=%v \n", err)
        return false
    }
    return true
}

func genRandomSeed(peerPublickKey * rsa.PublicKey) (bool, []byte, []byte){ 
    randomSeed :=RandByte()
    rng := rand.Reader
    encRandomSeed, err := rsa.EncryptOAEP(sha256.New(), rng, peerPublickKey, randomSeed, nil)
    if err != nil {
        log.Printf("EncryptOAEP failed, err=%v \n", err)
        return false, nil, nil
    }
    return true, randomSeed, encRandomSeed
}

func decPeerRandomSeed(myPrivateKey * rsa.PrivateKey, peerRandomSeed []byte) (bool, []byte) { 
    rng := rand.Reader
    decryptedPerRandomSeed, err := rsa.DecryptOAEP(sha256.New(), rng, myPrivateKey, peerRandomSeed, nil)
    if err != nil {
        log.Printf("DecryptOAEP failed, err=%v \n", err)
        return false, nil
    }

    return true, decryptedPerRandomSeed
}

func genSessionKey(myRandomDataToSign []byte, peerRandomDataToSign []byte, 
            myRandomSeed []byte, peerRandomSeed []byte) (bool, []byte) {
    var smallerDataToSign []byte
    var biggerDataToSign []byte
    var smallerSeed []byte
    var biggerSeed []byte
    if bytes.Compare(myRandomDataToSign, peerRandomDataToSign) <= 0 {
        smallerDataToSign = myRandomDataToSign
        biggerDataToSign = peerRandomDataToSign
	}  else {
        smallerDataToSign = peerRandomDataToSign
        biggerDataToSign = myRandomDataToSign
    }    
    
    if bytes.Compare(myRandomSeed, peerRandomSeed) <= 0 {
        smallerSeed = myRandomSeed
        biggerSeed = peerRandomSeed
	}  else {
        smallerSeed = peerRandomSeed
        biggerSeed = myRandomSeed
    }
    
    // session key=dhash(smaller  seed+bigger seed+smaller sign data+bigger sign data), iv=dhash(bigger seed+smaller seed)    
    buf := bytes.NewBuffer(smallerSeed)
    buf.Write(biggerSeed)
    buf.Write(smallerDataToSign)
    buf.Write(biggerDataToSign)
    sessionKey := dhash(buf.Bytes())
    return true, sessionKey
}

func getMyNickname() ([]byte){
    var buffer bytes.Buffer
    buffer.WriteString(RandString(4))
    buffer.WriteString(" ")
    buffer.WriteString(RandString(4))
    buffer.WriteString(" ")
    buffer.WriteString(RandString(4))
    
    // TODO: enc nickname, add add hash to last.
    return buffer.Bytes()
}

func verifingPeerNickname(recved []byte, sessionKey []byte) (bool){  
    return true
}

// 握手过程
func handshakeProcess(chid string, data []byte, bodyLen int, ctx * handshakeContext, autoAgreeUnknowFingerprint bool) {
    
    if handshakeStepSucessfully==ctx.handshakeStep {
        return
    }

    if handshakeStepWillSendMyVersion!=ctx.handshakeStep && bodyLen <= 0 {
        log.Printf("handshakeStep=%v, bodyLen=%v \n\n", ctx.handshakeStep, bodyLen)
        ctx.handshakeStep = handshakeStepFailed
        return
    }

    if handshakeStepWillSendMyVersion==ctx.handshakeStep {  
        // 发送我的版本号
        log.Printf("## handshakeStepWillSendMyVersion, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        ret, myVersion := getMyVersion()
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        if packAndSendHandshakeData(chid, myVersion) {
            ctx.handshakeStep =handshakeStepWaitPeerVersion
        } else {
            ctx.handshakeStep =handshakeStepFailed
        }
        
    } else if handshakeStepWaitPeerVersion==ctx.handshakeStep {  
        // 检查对方版本号
        log.Printf("## handshakeStepWaitPeerVersion, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        if false==verifingPeerVersion(data[:bodyLen]) {
            ctx.handshakeStep =handshakeStepFailed
            return
        }

        // 生成或读取我的公钥
        ret, myPrivateKey, myPublicKeyBuf :=getMyPublicKey()
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        ctx.myPrivateKey =myPrivateKey
        if packAndSendHandshakeData(chid, myPublicKeyBuf) {
            ctx.handshakeStep =handshakeStepWaitPeerPublicKey
        } else {
            ctx.handshakeStep =handshakeStepFailed
        }

    } else if handshakeStepWaitPeerPublicKey==ctx.handshakeStep {
        // 验证对方公钥
        log.Printf("## handshakeStepWaitPeerPublicKey, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        ret, peerPublickKey := checkPeerPublicKey(data[:bodyLen], autoAgreeUnknowFingerprint)
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        ctx.peerPublickKey =peerPublickKey

        // 生成签名用的随机数据发给对方
        ctx.myRandomDataToSign =genRandomDataToSign()
        if packAndSendHandshakeData(chid, ctx.myRandomDataToSign) {
            ctx.handshakeStep =handshakeStepWaitPeerRandomDataToSign
        } else {
            ctx.handshakeStep =handshakeStepFailed
        }

    } else if handshakeStepWaitPeerRandomDataToSign==ctx.handshakeStep {
        // 收到对方的随机数据, 用自己私钥签名后发给对方.
        log.Printf("## handshakeStepWaitPeerRandomDataToSign, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        ctx.peerRandomDataToSign = make([]byte, len(data[:bodyLen]))
        copy(ctx.peerRandomDataToSign, data[:bodyLen])
        ret, signedData := signPeerRandomData(ctx.peerRandomDataToSign, ctx.myPrivateKey)
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        
        if packAndSendHandshakeData(chid, signedData) {
            ctx.handshakeStep =handshakeStepWaitPeerSignedData
        } else {
            ctx.handshakeStep =handshakeStepFailed
        }
    
    }  else if handshakeStepWaitPeerSignedData==ctx.handshakeStep {
        // 用对方公钥来验证随机数的签名,无误后生成随机种子发给对方
        log.Printf("## handshakeStepWaitPeerSignedData, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        ret := verifingPeerSignedData(data[:bodyLen], ctx.peerPublickKey, ctx.myRandomDataToSign)
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        
        ret, randomSeed, encRandomSeed := genRandomSeed(ctx.peerPublickKey)
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        ctx.myRandomSeed =randomSeed
        
        if packAndSendHandshakeData(chid, encRandomSeed) {
            ctx.handshakeStep = handshakeStepWaitPeerRandomSeed
        } else {
            ctx.handshakeStep =handshakeStepFailed
        }
    
    } else if handshakeStepWaitPeerRandomSeed==ctx.handshakeStep {
        // 生成会话密钥
        log.Printf("## handshakeStepWaitPeerRandomSeed, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        
        ret, peerRandomSeed := decPeerRandomSeed(ctx.myPrivateKey, data[:bodyLen])
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        ctx.peerRandomSeed =peerRandomSeed;
        
        ret, sessionKey :=genSessionKey(ctx.myRandomDataToSign, ctx.peerRandomDataToSign,
             ctx.myRandomSeed, ctx.peerRandomSeed)
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        ctx.sessionKey =sessionKey        

        // 把我的昵称用会话密钥发给对方.
        myNickname :=getMyNickname()
        if packAndSendHandshakeData(chid, myNickname) {
            ctx.handshakeStep = handshakeStepWaitPeerNickname
        } else {
            ctx.handshakeStep =handshakeStepFailed
        }
    
    } else if handshakeStepWaitPeerNickname==ctx.handshakeStep {
        // 验证对方的昵称. 无误后握手成功.
        log.Printf("## handshakeStepWaitPeerNickname, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        ret := verifingPeerNickname(data[:bodyLen], ctx.sessionKey)
        if !ret {
            ctx.handshakeStep = handshakeStepFailed
            return
        }
        ctx.handshakeStep = handshakeStepSucessfully
        // log.Printf("##@@ handshakeStepSucessfully, chid=%v, handshakeStep=%v, sessionKey=%v \n\n", chid, ctx.handshakeStep, ctx.sessionKey)
        log.Printf("##@@ handshakeStepSucessfully, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        
    } else {
        log.Printf("##?? handshakeStepFailed, chid=%v, handshakeStep=%v \n\n", chid, ctx.handshakeStep)
        ctx.handshakeStep = handshakeStepFailed
    }

}
func GoID() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}

func readAndSendDataLoop(conn net.Conn, chid string, isServer bool, mode string, autoAgreeUnknowFingerprint bool, signalHandshakeOverQ chan string, signalMode2ToStartConnectQ chan bool) {

    pwd := " "
    if mode=="2" && isServer == false {
        pwd = <-signalHandshakeOverQ
        if len(pwd)<=1 {
            log.Printf("Recved handshake failed signal in clientHandler. mode:%v, isServer:%v \n\n", mode, isServer)
            return
        }
    }    
    if mode=="1" && isServer {
        pwd = <-signalHandshakeOverQ
        if len(pwd)<=1 {
            log.Printf("Recved handshake failed signal in serverHandler. mode:%v, isServer:%v \n\n", mode, isServer)
            return
        }
    }
        
    log.Printf("ENTER readAndSendDataLoop, recving data. chid:%v(%v), mode:%v, isServer:%v\n\n",
        chid, conn.RemoteAddr(), mode, isServer)

    var ctx handshakeContext
    encryptedSession := (mode == "1" && isServer == false) || (mode == "2" && isServer)
    if encryptedSession {
        ctx.handshakeStep = handshakeStepWillSendMyVersion
        // 连接上之后开始直接握手
        handshakeProcess(chid, nil, 0, &ctx, autoAgreeUnknowFingerprint)
        if handshakeStepFailed == ctx.handshakeStep {
            signalMode2ToStartConnectQ <- false
            signalHandshakeOverQ <- " "
            close(signalMode2ToStartConnectQ)
            close(signalHandshakeOverQ)
            removeConnQ <- chid
            return
        }
    }

	data := make([]byte, 1024*32)
	for {
        bodyLen :=0
        var err error
        if encryptedSession { // 加密的会话.
            bodyLen, err =readAndUnpackOnEncryptedSession(conn, chid, data)
            if err!=nil {
                break
            }
            
            if handshakeStepSucessfully!=ctx.handshakeStep {
                log.Printf("==>> Read handshake bodyLen:%v from %v(%v), mode:%v, isServer:%v\n\n",
                    bodyLen, chid, conn.RemoteAddr(), mode, isServer)
                    
                handshakeProcess(chid, data, bodyLen, &ctx, autoAgreeUnknowFingerprint)

                if handshakeStepFailed==ctx.handshakeStep { // 握手失败, 直接关闭连接.
                    signalMode2ToStartConnectQ <- false
                    signalHandshakeOverQ <- " "
                    close(signalMode2ToStartConnectQ)
                    close(signalHandshakeOverQ)
                    
                    removeConnQ <- chid
                    break

                } else if handshakeStepWaitPeerRandomDataToSign==ctx.handshakeStep {
                    signalMode2ToStartConnectQ <- true                 
                    continue
                
                } else if handshakeStepSucessfully==ctx.handshakeStep { // 握手成功, 清空中间数据.
                    ctx.myRandomDataToSign =nil
                    ctx.peerRandomDataToSign =nil
                    ctx.myRandomSeed =nil
                    ctx.peerRandomSeed =nil
                    
                    time.Sleep(1000 * time.Millisecond)
                    pwd = hex.EncodeToString(ctx.sessionKey)
                    signalHandshakeOverQ <- pwd
                    close(signalMode2ToStartConnectQ)
                    close(signalHandshakeOverQ)                    
                    continue

                } else { // 握手中, 继续下一步握手. 
                    continue
                }
            }
            
            
        } else { // 非加密的会话.
            bodyLen, err =readFullDataOnUnencryptedSession(conn, chid, data)
            if err!=nil {
                break
            }
        }
                
        if bodyLen <= 0 {
            log.Printf("bodyLen=%v \n\n", bodyLen)
            continue
        }
        recved := data[:bodyLen]
        
//        log.Printf("==>> Read data length:%v from %v(%v), mode:%v, isServer:%v\n\n",
//            len(recved), chid, conn.RemoteAddr(), mode, isServer)
            
        // mode
        // 0(Transparent): Transmit directly.
        // 1(Source):Encrypt accept's recved data, decrypt connect's recved data.
        // 2(Target): Decrypt accept's recved data, encrypt connect's recved data.
        if mode == "0" {
            sendQ <- sendQEle{chid, bytes.NewBuffer(recved)}
            // log.Printf("Read, transparent directly. Recved data:%v \n\n", string(recved))
            
        } else {
            if (mode == "1" && isServer) || (mode == "2" && isServer == false) {
                output, err := AESEncPacket(pwd, recved)
                if err != nil {
                    log.Printf("AESEncPacket failed, err=%v \n\n", err)
                    removeConnQ <- chid
                    break
                }
                
                // package
                b, err :=packData(output)
                if err != nil {
                    log.Printf("packData failed, err=%v \n\n", err)
                    removeConnQ <- chid
                    break
                }
                sendQ <- sendQEle{chid, b}
//                log.Printf("RECVED, encrypted , package, and send. Recved data:%v, sent data:%v \n\n", string(recved), output)
                
            } else if (mode == "1" && isServer == false) || (mode == "2" && isServer) {
                output, err := AESDecPacket(pwd, recved)
                if err != nil {
                    log.Printf("AESDecPacket failed, err=%v \n\n", err)
                    removeConnQ <- chid
                    break
                }
                
                sendQ <- sendQEle{chid, bytes.NewBuffer(output)}
//                log.Printf("RECVED, decrypted and send. Recved data:%v, sent data:%v \n\n", recved, string(output))
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

func contentOfFile(fileName string) ([]byte, error) {
    fin,err := os.Open(fileName)
    defer fin.Close()
    if err != nil {
        log.Printf("Open error. fileName=%v, error: %v \n\n", fileName, err)
        return nil, err
    }
    
    buf := make([]byte, 1024*16)
    total := 0
    for{
        n, err := fin.Read(buf[total:])
        if err != nil {
            if err==io.EOF {
                log.Printf("Read EOF. fileName=%v \n\n", fileName)
                total += n
                break
            } else {
                log.Printf("Read error. fileName=%v, error: %v \n\n", fileName, err)
                return nil, err
            }
        }
        total += n
        if 0 == n { break }
    }
    return buf[:total], nil
}

func writeToFile(fileName string, buf []byte) (error) {
    fout,err := os.Create(fileName)
    defer fout.Close()
    if err != nil {
        log.Printf("Create error. fileName=%v, error: %v \n\n", fileName, err)
        return err
    }
    
    total := 0
    for l :=len(buf);total<l; {
        n, err := fout.Write(buf[total:])
        if err != nil {
            log.Printf("Write error. fileName=%v, error: %v \n\n", fileName, err)
            return err
        }
        total += n
    }
    return nil
}

func appendToFile(fileName string, buf []byte) (error) {
    fout,err := os.OpenFile(fileName,os.O_APPEND|os.O_RDWR,0660)
    defer fout.Close()
    if err != nil {
        log.Printf("OpenFile error. fileName=%v, error: %v \n\n", fileName, err)

        fout, err = os.Create(fileName)  //创建文件
        if err != nil {
            log.Printf("Create error. fileName=%v, error: %v \n\n", fileName, err)
            return err
        } 
    }
    
    total := 0
    for l :=len(buf);total<l; {
        n, err := fout.Write(buf[total:])
        if err != nil {
            log.Printf("Write error. fileName=%v, error: %v \n\n", fileName, err)
            return err
        }
        total += n
    }
    return nil
}

func SetReadTimeout(conn net.Conn, timeoutSec int) {
	conn.SetDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second))
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

func RandByte() []byte {
	u1 := uuid.NewV4()
	u2 := uuid.NewV1()
	u := append(u1[0:], u2[0:]...)

	u3 := make([]byte, 32)
	_, err := rand.Read(u3)
	if err != nil {
		log.Printf("rand.Read failed, err=", err)
	} else {
		u = append(u, u3...)
	}
	return u
}

func RandString(length int) string {
	if length <= 0 {
		length = 4
	}
	u := make([]byte, length)
	_, err := rand.Read(u)
	if err != nil {
		log.Printf("rand.Read failed, err=", err)
		u1 := uuid.NewV4()
		u = u1[0:]
	}
	randStr := base32.StdEncoding.EncodeToString(u)
	return randStr[0:length]
}

func CalHash(src []byte, rounds int) []byte {
	dst := src
	for i := 0; i < rounds; i++ {
		t := sha256.Sum256(dst)
		dst = t[0:]
	}
	return dst
}

// 貌似实现有问题
func CalRipemd160(src []byte) []byte {
    h := ripemd160.New()
    h.Reset()
    h.Write(src)
	dst := h.Sum(nil)
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
    
//	log.Printf("AESEncPacket, pwd=%v, key=%v, iv=%v, input=%v, output=%v, err=%v", pwd, key, iv, input, output, err)
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
    
//	log.Printf("AESDecPacket, pwd=%v, key=%v, iv=%v, input=%v, output=%v, err=%v", pwd, key, iv, input, output, err)
	return
}

