# -*- coding: utf-8 -*-


import zokket
import time


class TCPDelegate(object):
    def __init__(self):

        zokket.Timer(1, self.onTimer, True)
        self.connected =-1  # -1 disconnected, 0 connecting, 1 connected.

        zokket.TCPSocket(self).connect(host='127.0.0.1', port=8601, timeout=4)
        self.connected =0
        print("Connecting to 127.0.0.1:8601 ...")

    def onTimer(self,timer):
        if -1==self.connected:
            zokket.TCPSocket(self).connect(host='127.0.0.1', port=8601, timeout=4)
            self.connected =0
            print("Reconnecting to 127.0.0.1:8601 ...")

    def socket_did_connect(self, sock, host, port):
        self.connected =1
        print('Connected to {}:{}'.format(host, port))
        sock.send('Hey!\n')

    def socket_connection_timeout(self, sock, host, port):
        self.connected =-1
        print("Connection to %s:%s timed out." % (host, port))

    def socket_did_disconnect(self, sock, err=None):
        self.connected =-1
        print("Disconnected ")

    def socket_read_data(self, sock, data):
        print('Received: {}'.format(data))

def main():
	TCPDelegate()
	zokket.DefaultRunloop.run()

if __name__ == '__main__':
	main()

