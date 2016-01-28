
##1 forwarding_server
forwarding_server有点像一个turn服务。forwarding_server会开启一个TCP服务，多个客户端连接到这个服务上，它负责转发任意两端(或多端)之间的tcp层报文, 所以这必然是一个神器。


运行在chan模式时，所有客户端连上服务握手成功以后会拿到通道标识即 chid， 握手过程中设置好接收者的chid列表，后续发给服务的所有数据都会被转发到设置的接收者那里。握手报文可以设置加密，负载数据是否加密服务不关心，它只是原样转发不作解析。

运行在broadcast模式时，所有客户端不需要与服务进行握手，连上服务后可以直接收发数据，服务原样广播给除发送者以外所有客户端。如果此时没有其它客户端连上服务，这些数据将会被丢弃。



####1.1 TODO:
```
1 超时关闭
2 自动生成chid
3 chan的缓冲等完善
4 架构完善
5 控制指令完成
```


####1.2 Client A want to send data to client B, the process is
```
client A [TCP stream encrypted by AES256]----> forwarding-server ----> [TCP stream encrypted by AES256] client B
```



####1.3 Concrete steps


***Step 1 , this is first step of handshake process.   Optional step when using broadcast mode.***

client send the following json to forwarding-server: 

```
{
    "req":"hs1",  // hs1 a.k.a first step of handshake process
    "chid":"ch0" // optional. Sever will generate if not set.
}
```


e.g.
client A send the following json to forwarding-server: 

```
{"req":"hs1","chid":"ch0"}
```

client B send the following json to forwarding-server:

```
 {"req":"hs1","chid":"ch1"}

```


***Step 2  , this is second step of handshake process.  Optional step when using broadcast mode.***

forwarding-server send back the following json to the client

```
{
    "rsp":"hs2",   
    "chid":"ch1" 
}

```


***Step 3  , this is third step of handshake process.  Optional step when using broadcast mode.***

client send the following json to forwarding-server: 

```
client send a json
{  
    "req":"hs3",   
    "recvers":["ch2","ch3","ch4"],   // optional, not set means broadcast to all others.
    "timeout":3600, // optinal, seconds
}

```


e.g.

client A: 

```
{"req":"hs3","recvers":["ch1"]}

```
client B:

```
 {"req":"hs3","recvers":["ch0"]}

```


***Step 4 , this is forth step of handshake process.  Optional step when using broadcast mode.***

forwarding-server send back the following json to the client

```
{   
    "rsp":"hs4",   
    "result":"OK" // "OK" means initial sucessfully, "FAIL" means initial failed.
    "desc":"" // optional.
}
```


***Step 5 , send data after handshake sucessfully***

client send data to forwarding-server who will foward the data to all receivers directly.


***Step 6 , close session when everthing is done***

client close the connection, then forwarding-server clear this session infomations.


## 2 forwarding_client

forwarding_client只支持发起两个向外TCP连接(称为连接1和连接2)，内部不会侦听任何端口。forwarding_client会把在连接1上收到的数据原样转发到连接2上，反之，在连接2上收到的数据会原样转发到连接1上。 这个过程不会对数据作任何修改。 与forwarding_server的区别是，forwarding_client是转发两个TCP服务器的数据，而forwarding_server是转发两个(或多个)TCP客户端之间的数据。

##3 使用示例

*  forwarding_server 使用

```
$go run forwarding_server.go
2016/01/11 19:53:24 -- Main started. Listen addr::8601, handshake password:, mode:broadcast
```
    
*  forwarding_client 使用

```
$go run forwarding_client.go
2016/01/11 19:53:31 -- Main started. Server0 addr:127.0.0.1:8601,  Server1 addr:127.0.0.1:8080
```

##4 author

author: jeffery

email: dungeonsnd at gmail dot com


