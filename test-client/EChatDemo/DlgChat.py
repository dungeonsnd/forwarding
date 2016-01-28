# -*- coding: utf-8 -*-

"""
Module implementing DlgChat.
"""

from PyQt4 import QtCore
from PyQt4 import QtGui
import os.path

from Ui_DlgChat import Ui_Dialog

import zokket
import time
import pack
import urllib2

print_log =False
send_connected_info =True

class DlgChat(QtGui.QDialog, Ui_Dialog):
    """
    Class documentation goes here.
    """
    def __init__(self, parent=None):
        """
        Constructor
        
        @param parent reference to the parent widget (QWidget)
        """
        self.sock =None
        self.host =''
        self.port =443
        self.chid =''
        self.pwd =''
        self.url ='https://raw.githubusercontent.com/dungeonsnd/myexternalresources/master/rpi_ip_encrypted/ip.txt'
        
        self.connected =False
        self.join_time =''
        self.all_chids ={} # chid->join_time
        
        self.opacity =0.95
        
        QtGui.QDialog.__init__(self, parent)
        self.setupUi(self)
        self.setWindowTitle(u'EChat v0.9  ----来自开发者小杰 dungeonsnd@gmail.com')
        self.setWindowFlags(QtCore.Qt.Window)
        self.setWindowFlags(QtCore.Qt.WindowMinimizeButtonHint)
        self.setFixedSize(self.width(), self.height());
        
        self.setWindowOpacity(self.opacity)
        #self.color=QtGui.QColor(60, 80, 80)
        #self.setStyleSheet('QWidget{background-color:%s}'%self.color.name())
        
        self.editInfo.append(u"EChat会建立安全通道, 能够确保你的聊天消息不会任何个人或组织窃取! 包括软件作者及服务器端也无法获知你的聊天内容!")
        self.editInfo.append(u"Windows 最新版下载 https://github.com/dungeonsnd/forwarding/raw/master/test-client/EChatDemo/dist/EChat-win.rar")
        self.editInfo.append(u"Mac OSX 最新版下载 https://github.com/dungeonsnd/forwarding/raw/master/test-client/EChatDemo/dist/EChat-osx.tar.xz")
        self.editInfo.append(u"请输入 加密密钥和用户名称，然后单击连接服务，等待连接成功后就可以加入群聊。服务器不填使用默认服务器。")
        self.editInfo.append(u"注意, 加密密钥必须较长且复杂不易被人猜测到，否则猜到密码就可以加入你的群聊中!  Good day! ")

    def keyPressEvent(self, e):
        if QtCore.Qt.Key_Escape==e.key():
            self.close()
        if QtCore.Qt.Key_Return==e.key():
            self.sendMsg()
    
    # 每次单击改变窗口透明度    
    def mousePressEvent(self, e):
        self.opacity = self.opacity-0.3
        #print 'self.opacity=', self.opacity
        if self.opacity<0.01:
            self.opacity =0.95
        self.setWindowOpacity(self.opacity)
        
    def closeEvent(self, event):
        reply = QtGui.QMessageBox.question(self, u'警告',
            u"退出程序?", QtGui.QMessageBox.Yes | QtGui.QMessageBox.No, 
            QtGui.QMessageBox.Yes)
        if reply == QtGui.QMessageBox.Yes:
            self.sendDic( { 'cmd':'client_leave',
                'chid':self.chid,
                'timenow':self.timeNow() } )
            if self.sock:
                self.sock.close()
            event.accept()
        else:
            event.ignore()

    def timeNow(self):
        return time.strftime('%Y-%m-%d %H:%M:%S',time.localtime(time.time()))
        
    def sendDic(self, dic):
        if print_log:
            print 'DlgChat::sendDic=', dic
        output =pack.pack(self.pwd, dic)
        if len(output)>0 and self.sock:
            try:                
                self.sock.send(output)
            except :
                self.sock.close()
                self.connected =False
                self.join_time =''
                self.editInfo.append(u'服务器已断连，请重新连接服务!');
                return False
            else:
                return True
        else:
            return False

    def sendMsg(self):
        if not self.sock:
            self.editInfo.append(u"请先连接服务器！");
            QtGui.QMessageBox.information( self, u'信息', u'请先连接服务器！' )
            return 
            
        txtUnicode =self.editInput.toPlainText()        
        timenow =self.timeNow()
        dic ={'cmd':'msg', 
            'chid':self.chid,  
            'txt':str(txtUnicode.toUtf8()),  
            'timenow':timenow}
        if not self.sendDic(dic):
            self.editInfo.append(u"发送失败！");
            return

        self.editOutput.append('['+timenow +'] '+self.chid.decode('utf-8') +u' (自己)说: '+ txtUnicode)
        self.editInput.clear()

    def socket_connection_timeout(self, sock, host, port):
        self.connected =False
        self.join_time =''
        self.editInfo.append(u'连接服务 %s:%s 超时! ' % (host, port))

    def socket_did_connect(self, sock, host, port):
        self.editInfo.append(u'已经连接上服务 %s:%d'%(self.host, self.port))
        self.connected =True
        self.sock =sock
        sock.read_until_data = '\r\n'
        timenow =self.timeNow()
        self.join_time =timenow
        
        if send_connected_info:
            self.sendDic( { 'cmd':'new_client_join',
            'chid':self.chid,
            'timenow':timenow } )
            self.editOutput.append(u'[%s] [系统消息] 欢迎 %s 加入聊天'%(timenow, self.chid.decode('utf-8')))
    def socket_did_secure(self, sock):
        self.editInfo.append(u'enter socket_did_secure. ')
    
    def onRecvedFile(self, d, chid_utf8, timenow):
        reply = QtGui.QMessageBox.question(self, u'通知',
            u'%s发送了%s字节的文件 %s，是否接收?'%(chid_utf8.decode('utf-8'), d['filesize'], d['filename']), 
            QtGui.QMessageBox.Yes | QtGui.QMessageBox.No, QtGui.QMessageBox.Yes)
        if reply == QtGui.QMessageBox.Yes:
            dic ={'cmd':'recvfile', 'chid':self.chid,
                'sender_chid':chid_utf8, 
                'filename':d['filename'].encode('utf-8'),  'txt':'',  'timenow':self.timeNow()}
        else:
            dic ={'cmd':'rejectfile', 'chid':self.chid,
                'sender_chid':chid_utf8,
                'filename':d['filename'].encode('utf-8'),  'txt':'',  'timenow':self.timeNow()}
        self.sendDic(dic)

    def onNewClientJoin(self, chid_utf8, timenow):
        # 保存新加入的人
        self.all_chids[chid_utf8] =timenow            
        # 通知新加入的人自己在线
        self.sendDic({'cmd':'already_online_client', 
            'join_time':self.join_time,
            'new_join_chid': chid_utf8,
            'chid':self.chid, 
            'timenow':self.timeNow()})        
        self.editOutput.append(u'[%s] [系统消息] %s 加入了聊天'%(timenow, chid_utf8.decode('utf-8')))
        
    def socket_read_data(self, sock, data):
        # zokket收到的是unicode编码 ,转换为utf-8再进行json解析。
        input =data.encode('utf-8')
        d =pack.unpack(self.pwd, input)
        if print_log:
            print 'typeof unpack result:', type(d)
            print 'unpack input=', input
            print 'unpack output=', d
        if len(d)<1:
            return
        if type(d) != type({}):
            return
        # 注意json解析结果是unicode编码 !
        cmd =d['cmd']
        chid =d['chid']
        timenow =d['timenow']
        
        chid_utf8 =chid.encode('utf-8')
        flicker =True 
        if u'new_client_join'==cmd: # 已经在线的人 收到新加入报文，发出自己在线的通知
            self.onNewClientJoin(chid_utf8, timenow)
        if u'client_leave'==cmd: # 已经在线的人 收到其它人离线报文
            if self.all_chids.has_key(chid_utf8):
                del(self.all_chids[chid_utf8])
                self.editOutput.append(u'[%s] [系统消息] %s 离开了聊天'%(timenow, chid))
            
        elif u'already_online_client'==cmd: # 新加入的人(以及其它人) 收到在线通知，只有新加入的人才处理. 
            if d['new_join_chid'].encode('utf-8')==self.chid:         
                self.all_chids[chid_utf8] =d['join_time']# 保存在线的人
                
        elif u'sendfile'==cmd: # 收到文件通知
            self.onRecvedFile(d, chid_utf8, timenow)
        elif u'recvfile'==cmd: # 接收文件通知，只有发送者才处理这条报文
            if str(d['sender_chid'].encode('utf-8'))==self.chid:
                QtGui.QMessageBox.information( self, u'系统通知', u'%s 接受了文件%s'%(chid,  d['filename']) )
        elif u'rejectfile'==cmd: # 拒接文件通知，只有发送者才处理这条报文
            if str(d['sender_chid'].encode('utf-8'))==self.chid:
                QtGui.QMessageBox.information( self, u'系统通知', u'%s 拒接了文件%s'%(chid,  d['filename']) )   
                
        elif u'msg'==cmd: # 收到消息
            self.editOutput.append(u'[%s] %s 说: %s'%(timenow, chid, d['txt']))
        else : # 其它消息
            flicker =False

        if flicker:
            QtGui.QApplication.alert(self, 0) # windows任务栏闪烁提醒.   
   
    def socket_did_disconnect(self, sock, err=None):
        self.connected =False
        self.join_time =''
        self.editInfo.append(u'Disconnected ')
        self.editInfo.append(u'服务 %s:%s 已断开'% (self.host.decode('utf-8'), self.port))
        
    def getHostFromGithub(self):
        html =urllib2.urlopen( self.url ).read()
        idx1 =html.index('external=')
        idx2 =html.index('local=')
        s =html[idx1+len('external=')+1:idx2-1]                    
        ip ='.'.join( str(int(x)-10001) for x in s.split('.') )
        #print 'get ip from github is:', ip
        self.editInfo.append(u"获取服务ip成功 %s"%(ip.decode('utf-8')))
        if len(ip)>0:
            self.host =ip
            self.port =443
        
    @QtCore.pyqtSignature("")
    def on_btnConnectSvr_clicked(self):
        """
        Slot documentation goes here.
        """
        if self.connected:
            return
            
        pwd =self.editPwd.text()
        if len(pwd)<1:
            self.editInfo.append(u'加密密钥不能为空！');
            return
        self.pwd =str(pwd.toUtf8())
        
        chid =self.editChid.text()
        if len(chid)<1:
            self.editInfo.append(u'用户名称不能为空！');
            return
        self.chid =str(chid.toUtf8())
        #print 'self.chid=', self.chid
            
        svrAddr=str(self.editSvrAddr.text().toUtf8())
        if len(svrAddr)>0 :
            r =svrAddr.split(':')
            if len(r)>0:
                self.host =r[0]
            if len(r)>1:
                self.port =int(r[1])
            self.editInfo.append(u"使用输入的服务地址, %s:%d"%(self.host.decode('utf-8'), self.port))
        else:
            self.host =''

        if len(self.host)<1 or self.port<1 : # Use default.
            self.editInfo.append(u"正在获取服务ip, 请稍候...")
            try:
                self.getHostFromGithub()
            except:
                self.editInfo.append(u"获取服务地址失败, %s:%d"%(self.host, self.port))
            else:
                zokket.TCPSocket(self).connect(host=self.host, port=self.port, timeout=6)
                self.editInfo.append(u"正在连接 %s:%d ..."%(self.host, self.port))        
                self.btnConnectSvr.enable =False;
    
    
    @QtCore.pyqtSignature("")
    def on_btnSend_clicked(self):
        """
        Slot documentation goes here.
        """
        self.sendMsg()
    
    @QtCore.pyqtSignature("")
    def on_btnContactList_clicked(self):
        """
        Slot documentation goes here.
        """
        # 显示在线的人
        ss =u''
        if len(self.all_chids)>0:
            for (i, v) in self.all_chids.items():
                ss =ss+u'%s 加入时间:%s \n'%(i.decode('utf-8'), v.decode('utf-8'))
        if len(ss)>0:            
            QtGui.QMessageBox.information( self, u'在线列表', ss )
        else:
            QtGui.QMessageBox.information( self, u'在线列表', u'当前没有其它人在线' )
            
    @QtCore.pyqtSignature("")
    def on_btnSendFile_clicked(self):
        """
        Slot documentation goes here.
        """
        if not self.sock:
            self.editInfo.append(u"请先连接服务器！");
            return
            
        s = QtGui.QFileDialog.getOpenFileName(None, u"open file dialog")
        if len(s)>0:
            fsize =os.path.getsize(s)
            d ={'cmd':'sendfile', 
                'chid':self.chid, \
                'filename':str(s.toUtf8()), 
                'filesize':str(fsize),  
                'timenow':self.timeNow()}
            output =pack.pack(self.pwd, d)
            if len(output)<1:
                self.editInfo.append(u"发送文件失败！");
            else:
                self.sock.send(output)
    
    @QtCore.pyqtSignature("")
    def on_btnSetting_clicked(self):
        """
        Slot documentation goes here.
        """
        QtGui.QMessageBox.information( self, u'信息', u'暂时不支持该功能' )
    
    @QtCore.pyqtSignature("")
    def on_btnMediaMeeting_clicked(self):
        """
        Slot documentation goes here.
        """
        QtGui.QMessageBox.information( self, u'信息', u'暂时不支持该功能' )
