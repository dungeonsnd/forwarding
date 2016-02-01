# -*- coding: utf-8 -*-


import zokket
import time


class TCPDelegate(object):
    def __init__(self):
        zokket.TCPSocket(self).connect(host='127.0.0.1', port=8601)

    def socket_did_connect(self, sock, host, port):
        print('connected to {}:{}'.format(host, port))
        sock.send('Hey!\n')

    def socket_read_data(self, sock, data):
        print('Received: {}'.format(data))


    def socket_connection_timeout(self, sock, host, port):
        print("Connection to %s:%s timed out." % (host, port))

    def socket_did_disconnect(self, sock, err=None):
        print("Disconnected ")
        time.sleep(8)
        print('reonnectting...\n')
        zokket.TCPSocket(self).connect(host='127.0.0.1', port=8601)

def main():
	TCPDelegate()
	zokket.DefaultRunloop.run()

if __name__ == '__main__':
	main()
