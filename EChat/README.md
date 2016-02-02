

### EChat   是一个群聊客户端
使用AES加密聊天内容，基本功能初步完成。可以使用默认服务，在客户端的服务器一栏不填即可，这时使用作者的一个树莓派单板机作为服务器。 另外，你也可以自己启一个服务， 服务端请使用
 [https://github.com/dungeonsnd/forwarding/blob/master/forwarding/forwarding_server.go](https://github.com/dungeonsnd/forwarding/blob/master/src/forwarding_server.go)  ，可运行于WIN/OSX/LINUX。注意如果不用默认服务而是自己启 forwarding_server或者自己重新实现服务端的话，它需要在公网启动，或者在内网启动后做端口映射以便其它客户端可以访问。


### 二进制可执行程序下载地址

Windows 最新版下载 [https://github.com/dungeonsnd/forwarding/raw/master/EChat/dist/EChat-win.rar](https://github.com/dungeonsnd/forwarding/raw/master/EChat/dist/EChat-win.rar)

Mac OSX 最新版下载 [https://github.com/dungeonsnd/forwarding/raw/master/EChat/dist/EChat-osx.tar.xz](https://github.com/dungeonsnd/forwarding/raw/master/EChat/dist/EChat-osx.tar.xz)


### 等待完成列表  ( TODO List )

* 重连有问题。 MAC版本过几个小时显示连接已断开，这里重连连不上了。测试发现，重连时服务端显示连接上了，但是端上没进入连接成功回调，而是过一会进入连接超时回调。怀疑是 zokket问题，有待解决。
Win版本过一段时间也会显示连接断开，但是可以直接点击界面上的重连连上服务器。

* 把网络相关的部分独立出来 （工作量：小）
* 文件收发支持 （小）
* Ubuntu下打包成执行程序 （中）
* UPnP及p2p支持 （大）
* 音视频支持 （大）
* 获取所有可用服务器列表功能，任意选择某一服务器 （小）
* 文件共享功能 （中）
* iOS/Anroid 客户端支持 （大）
* 终端基本聊天支持。使用加密和网络等基本的封装模块  (小)
* 界面完善及UI美化 （中）



### 开发环境
开发：

Win7 
Eric6-6.0.5  
PyQt GPL v4.11.3 for Python 2.7 (x32)

Win打包环境: 
py2exe

MAC打包环境：
brew install pyqt
py2app 


### 打包相关的文件

dist 是打包工具生成二进制发行包的目录
py2exe_package.bat 用于WINDOWS环境下打包
py2app_package.sh 用于MAC环境下打包
cx-freeze_package.sh.failed 是测试失败的打包脚本，后面如果接着在LINUX发行版下打包可能会用得着它。
ls.ico是程序图标



