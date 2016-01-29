
* EChatDemo 一个群聊客户端，使用AES加密聊天内容。基本功能初步完成。 服务端请使用 https://github.com/dungeonsnd/forwarding/blob/master/src/forwarding_server.go  ，可运行于WIN/OSX/LINUX。可以使用默认服务，在客户端的服务器一栏不填即可，这时使用作者的一个树莓派单板机作为服务器。 另外，你也可以自己启一个服务，注意需要在公网起服务，或者在内网启动后做端口映射以便其它客户端可以访问。

二进制可执行程序下载地址:

Windows 最新版下载 https://github.com/dungeonsnd/forwarding/raw/master/test-client/EChatDemo/dist/EChat-win.tar.xz

Mac OSX 最新版下载 https://github.com/dungeonsnd/forwarding/raw/master/test-client/EChatDemo/dist/EChat-osx.tar.xz



开发环境是

Win7

Eric6-6.0.5

PyQt GPL v4.11.3 for Python 2.7 (x32)


打包想着的文件如下：

dist 是打包工具生成二进制发行包的目录

py2exe_package.bat 用于WINDOWS环境下打包

py2app_package.sh 用于MAC环境下打包

cx-freeze_package.sh.failed 是测试失败的打包脚本，后面如果接着在LINUX发行版下打包可能会用得着它。

ls.ico是程序图标
