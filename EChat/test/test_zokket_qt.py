# -*- coding: utf-8 -*-

import sys

from PyQt4 import QtCore
from PyQt4 import QtGui
from PyQt4 import QtWebKit

from zokket.qt import QtRunloop
import zokket
import time

class Window(QtGui.QMainWindow):
    def __init__(self):
        super(Window, self).__init__()

        self.setWindowTitle('zokket - PyQT Example')

        self.editor = QtGui.QTextEdit()
        self.editor.setReadOnly(True)
        self.setCentralWidget(self.editor)
        self.resize(500, 300)

        self.show()

        # create timer
        self.timer=QtCore.QTimer()
        QtCore.QObject.connect(self.timer,QtCore.SIGNAL("timeout()"), self.onTimer)
        self.timer.start(1000)
        self.connected =-1  # -1 disconnected, 0 connecting, 1 connected.

        zokket.TCPSocket(self).connect(host='127.0.0.1', port=8601, timeout=4)
        self.connected =0
        self.editor.append("Connecting to 127.0.0.1:8601...")
        
    def onTimer(self):
        if -1==self.connected:
            zokket.TCPSocket(self).connect(host='127.0.0.1', port=8601, timeout=4)
            self.connected =0
            self.editor.append("Connecting to 127.0.0.1:8601 ...")


    # Socket delegate methods

    def socket_did_connect(self, sock, host, port):
        self.editor.append("Connected")
        self.connected =1

    def socket_did_disconnect(self, sock, err=None):
        self.editor.append("\r\nDisconnected")
        self.connected =-1

    def socket_connection_timeout(self, sock, host, port):
        self.editor.append("Connection to %s:%s timed out." % (host, port))
        self.connected =-1

    def socket_read_data(self, sock, data):
        self.editor.append("> " + data.strip())


if __name__ == '__main__':
    app = QtGui.QApplication(sys.argv)

    # Tell zokket to use the QtRunloop
    QtRunloop.set_default(app)

    # Start our window
    Window()

    sys.exit(app.exec_())

