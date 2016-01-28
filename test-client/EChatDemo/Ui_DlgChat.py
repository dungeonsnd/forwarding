# -*- coding: utf-8 -*-

# Form implementation generated from reading ui file 'E:\mydoc\mycode\pyqt\EChatDemo\DlgChat.ui'
#
# Created: Wed Jan 27 15:41:22 2016
#      by: PyQt4 UI code generator 4.11.3
#
# WARNING! All changes made in this file will be lost!

from PyQt4 import QtCore, QtGui

try:
    _fromUtf8 = QtCore.QString.fromUtf8
except AttributeError:
    def _fromUtf8(s):
        return s

try:
    _encoding = QtGui.QApplication.UnicodeUTF8
    def _translate(context, text, disambig):
        return QtGui.QApplication.translate(context, text, disambig, _encoding)
except AttributeError:
    def _translate(context, text, disambig):
        return QtGui.QApplication.translate(context, text, disambig)

class Ui_Dialog(object):
    def setupUi(self, Dialog):
        Dialog.setObjectName(_fromUtf8("Dialog"))
        Dialog.resize(987, 655)
        font = QtGui.QFont()
        font.setPointSize(11)
        Dialog.setFont(font)
        Dialog.setWindowTitle(_fromUtf8(""))
        Dialog.setSizeGripEnabled(True)
        self.btnSend = QtGui.QPushButton(Dialog)
        self.btnSend.setGeometry(QtCore.QRect(770, 580, 211, 71))
        font = QtGui.QFont()
        font.setPointSize(16)
        self.btnSend.setFont(font)
        self.btnSend.setObjectName(_fromUtf8("btnSend"))
        self.editInput = QtGui.QTextEdit(Dialog)
        self.editInput.setGeometry(QtCore.QRect(10, 520, 751, 131))
        font = QtGui.QFont()
        font.setPointSize(16)
        self.editInput.setFont(font)
        self.editInput.setObjectName(_fromUtf8("editInput"))
        self.editOutput = QtGui.QTextEdit(Dialog)
        self.editOutput.setGeometry(QtCore.QRect(10, 130, 751, 381))
        font = QtGui.QFont()
        font.setPointSize(16)
        self.editOutput.setFont(font)
        self.editOutput.setReadOnly(True)
        self.editOutput.setObjectName(_fromUtf8("editOutput"))
        self.editInfo = QtGui.QTextEdit(Dialog)
        self.editInfo.setGeometry(QtCore.QRect(10, 0, 751, 121))
        font = QtGui.QFont()
        font.setFamily(_fromUtf8("SimSun-ExtB"))
        font.setPointSize(14)
        font.setBold(False)
        font.setWeight(50)
        self.editInfo.setFont(font)
        self.editInfo.setVerticalScrollBarPolicy(QtCore.Qt.ScrollBarAlwaysOn)
        self.editInfo.setHorizontalScrollBarPolicy(QtCore.Qt.ScrollBarAlwaysOff)
        self.editInfo.setReadOnly(True)
        self.editInfo.setObjectName(_fromUtf8("editInfo"))
        self.editPwd = QtGui.QLineEdit(Dialog)
        self.editPwd.setGeometry(QtCore.QRect(770, 30, 211, 31))
        self.editPwd.setEchoMode(QtGui.QLineEdit.Password)
        self.editPwd.setObjectName(_fromUtf8("editPwd"))
        self.editSvrAddr = QtGui.QLineEdit(Dialog)
        self.editSvrAddr.setEnabled(True)
        self.editSvrAddr.setGeometry(QtCore.QRect(770, 170, 211, 31))
        self.editSvrAddr.setObjectName(_fromUtf8("editSvrAddr"))
        self.btnConnectSvr = QtGui.QPushButton(Dialog)
        self.btnConnectSvr.setGeometry(QtCore.QRect(770, 220, 211, 41))
        self.btnConnectSvr.setObjectName(_fromUtf8("btnConnectSvr"))
        self.editChid = QtGui.QLineEdit(Dialog)
        self.editChid.setGeometry(QtCore.QRect(770, 100, 211, 31))
        self.editChid.setObjectName(_fromUtf8("editChid"))
        self.label = QtGui.QLabel(Dialog)
        self.label.setGeometry(QtCore.QRect(770, 10, 221, 21))
        self.label.setObjectName(_fromUtf8("label"))
        self.label_2 = QtGui.QLabel(Dialog)
        self.label_2.setGeometry(QtCore.QRect(770, 80, 71, 20))
        self.label_2.setObjectName(_fromUtf8("label_2"))
        self.label_3 = QtGui.QLabel(Dialog)
        self.label_3.setEnabled(True)
        self.label_3.setGeometry(QtCore.QRect(770, 150, 61, 20))
        self.label_3.setObjectName(_fromUtf8("label_3"))
        self.btnContactList = QtGui.QPushButton(Dialog)
        self.btnContactList.setEnabled(True)
        self.btnContactList.setGeometry(QtCore.QRect(770, 280, 211, 41))
        self.btnContactList.setObjectName(_fromUtf8("btnContactList"))
        self.btnSendFile = QtGui.QPushButton(Dialog)
        self.btnSendFile.setEnabled(True)
        self.btnSendFile.setGeometry(QtCore.QRect(770, 340, 211, 41))
        self.btnSendFile.setObjectName(_fromUtf8("btnSendFile"))
        self.btnSetting = QtGui.QPushButton(Dialog)
        self.btnSetting.setGeometry(QtCore.QRect(770, 460, 211, 41))
        self.btnSetting.setObjectName(_fromUtf8("btnSetting"))
        self.btnMediaMeeting = QtGui.QPushButton(Dialog)
        self.btnMediaMeeting.setGeometry(QtCore.QRect(770, 400, 211, 41))
        self.btnMediaMeeting.setObjectName(_fromUtf8("btnMediaMeeting"))

        self.retranslateUi(Dialog)
        QtCore.QMetaObject.connectSlotsByName(Dialog)

    def retranslateUi(self, Dialog):
        self.btnSend.setText(_translate("Dialog", "发送(Ctrl+Enter)", None))
        self.editPwd.setPlaceholderText(_translate("Dialog", "尽量包含标点符号及大写字母", None))
        self.editSvrAddr.setPlaceholderText(_translate("Dialog", "可不填(连默认服务).ip:port", None))
        self.btnConnectSvr.setText(_translate("Dialog", "连接服务", None))
        self.editChid.setPlaceholderText(_translate("Dialog", "任意字符，作为聊天中昵称", None))
        self.label.setText(_translate("Dialog", "加密密钥", None))
        self.label_2.setText(_translate("Dialog", "用户名称", None))
        self.label_3.setText(_translate("Dialog", "服务器", None))
        self.btnContactList.setText(_translate("Dialog", "联系人列表", None))
        self.btnSendFile.setText(_translate("Dialog", "发送文件", None))
        self.btnSetting.setText(_translate("Dialog", "设置", None))
        self.btnMediaMeeting.setText(_translate("Dialog", "音视频会议", None))


if __name__ == "__main__":
    import sys
    app = QtGui.QApplication(sys.argv)
    Dialog = QtGui.QDialog()
    ui = Ui_Dialog()
    ui.setupUi(Dialog)
    Dialog.show()
    sys.exit(app.exec_())

