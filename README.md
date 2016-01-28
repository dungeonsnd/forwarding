# forwarding
A very liteweight tool to forward data over tcp,  written in Go.


## 1 主要功能
本项目旨在实现下面几个功能，即

* **src/forwarding_server.go    用于透明转发多个任意客户端之间的数据流**。运行后它会侦听一个TCP端口，多个客户端连上这个服务，这些客户端可以相互收发数据，数据对这个服务来说是无协议的。  你可以选择给部分客户端发数据，这种情况下需要在建立连接后与服务进行一次握手过程； 你也可以选择不进行握手而是连上之后直接收发数据，这种情况下，由于你没有握手，所以服务也不会知道你要发给哪一个客户端，所以会在全部客户端中进行广播，有点像一个群聊模型。  **本服务的核心功能已经基本完成，可用于非苛刻条件下的生产环境**。


* **src/forwarding_client.go   它负责转发任意两个服务之间的数据流**。本身不侦听TCP端口，而是作为客户端连上指定的TCP服务。它不关心协议，也没有握手过程。**本服务的核心功能已经基本完成，可用于非苛刻条件下的生产环境**。


* **src/forwarding_proxy.go**  这纯粹是无聊的玩意，感觉有了服务端转发和客户端转发，没有一个代理总是有点强迫。 这东西有其它现成的可以用，如 ssh 端口转发、sock5代理等等。 而且，咱这个只是个玩具，原样转发数据，没有别人的强大，不过可以偶尔用用。**本服务的还未完成，暂时不要使用**。


forwarding_server 相当于多个"母"接口， forwarding_client 相当于两个"公"接口。 你可以把"母"接口放在公网，把局域网中运行的 forwarding_client 的其中一个"公"接口接到母接口上，另一个"公"接口连上局域网的某个TCP服务，这样就可以在其它网络访问这个局域网的那个TCP服务。注意，这一功能不能用于HTTP透明传输服务，因为forwarding_client只暴露两个"公"接口，而访问某个网站时通常需要很多的公接口。如果要支持HTTP类似的多连接，就需要让 forwarding_client 支持两个以上的"公"接口，也就是需要连接池，作者暂时没有计划去实现该功能。


具体还能干嘛? 你懂的，嘿嘿 ^=^


## 2 示例程序

不要光看src下的主要程式哦，示例程序也有惊喜哦 ^_^

比如 test-client/EChatDemo 实现了一个聊天应用，已经可以用于普通文本聊天，图片聊天、文件收发、音视频会议正在开发集成中。服务端使用 src/forwarding_server.go 即可。 如果只是想使用，可以下载可执行程序，目前提供windows和mac两种系统的版本，绿色无污染。


可以直接下载二进制程序试玩一下哦:

Windows 最新版下载 https://github.com/dungeonsnd/forwarding/raw/master/test-client/EChatDemo/dist/EChat-win.rar

Mac OSX 最新版下载 https://github.com/dungeonsnd/forwarding/raw/master/test-client/EChatDemo/dist/EChat-osx.tar.xz


## 3 加星

喜欢的一定要给个 star 啊， 求星星， 求赞!   阁下的支持是作者们最大的动力!


## 4 author

author: jeffery

email: dungeonsnd at gmail dot com


