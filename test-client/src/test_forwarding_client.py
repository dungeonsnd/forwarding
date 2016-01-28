#!/bin/env python
#-*- encoding: utf-8 -*-

import sys
import socket
import Queue
import json
import time
import threading

## private
sock=None

def thread_recver():
    global sock

    while True:
        try:
            data = sock.recv(8192)
            if not data:
                break
            print '#### Received', repr(data)
        except Exception, e:
            break
        else:
            pass

def Start():
    t =threading.Thread(target=thread_recver, args=())
    t.start()
    return t

def Run(bindPort):
    global sock
    HOST = ''
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind((HOST, bindPort))
    s.listen(10)
    sock, addr = s.accept()

    t =Start()

    print 'Connected by', addr
    while 1:
        try:
            inputData = raw_input("")
            sock.sendall(inputData)
        except Exception, e:
            print e
            break
    conn.close()

if __name__ == '__main__':
    if len(sys.argv) <2:
        print '%s <bind-port>'%(sys.argv[0])
        print " e.g. python %s 8080"%(sys.argv[0])
        sys.exit(0)
    bindPort =int(sys.argv[1])
    Run(bindPort)
