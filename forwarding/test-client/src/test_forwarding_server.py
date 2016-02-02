#!/bin/env python
#-*- encoding: utf-8 -*-

import sys
import socket
import Queue
import json
import time
import threading

# public
serverIp ='127.0.0.1'
serverPort =8601
chid=None
recvQueue = Queue.Queue() 
sendQueue = Queue.Queue()


## private
sock=None
sockExceptionEvent = threading.Event()
stopThread =False

def thread_connector():
    global chid
    global sock
    global sockExceptionEvent
    global stopThread

    HOST = serverIp
    PORT = serverPort
    print 'chid=',chid
    if len(chid)<1:
        return

    sockExceptionEvent.clear()
    while True:
        try:
            if sock:
                sock.close()
            sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            sock.connect((HOST, PORT))
            print 'sock connected'

            # handshake
            # step 1
            d ={"req":"hs1","chid":chid}
            sock.sendall(json.dumps(d)+'\n')
            print 'step 1, sent ', json.dumps(d)+'\n'

            # step 2
            data = sock.recv(8192)
            if not data:
                sockExceptionEvent.set()
                break
            d =json.loads(data)
            print 'step2, received ', d

            # step 3
            d ={"req":"hs3"}
            sock.sendall(json.dumps(d)+'\n')
            print 'step 3, sent ', json.dumps(d)+'\n'

            # step 4
            data = sock.recv(8192)
            if not data:
                sockExceptionEvent.set()
                break
            d =json.loads(data)
            print 'step4, received ', d

            # after handshaked, start sender and recver.
            t1 =threading.Thread(target=thread_sender, args=())
            t2 =threading.Thread(target=thread_recver, args=())
            t1.start()
            t2.start()

            # reconnect when recv socket exception
            sockExceptionEvent.wait()
            sockExceptionEvent.clear()

            stopThread =True

            t1.join()
            t2.join()
        except Exception, e:
            print e
            time.sleep(2.0)
            continue
        else:
            pass

def thread_sender():
    global sock
    global sendQueue
    global sockExceptionEvent
    global stopThread

    while not stopThread:
        try:
            item = sendQueue.get(False,1)
            sock.sendall(item)
            print '==== sendall,len=%d'%(len(item))
        except Queue.Empty, e:
            continue
        except Exception, e:
            sockExceptionEvent.set()
            break
        else:
            pass

def thread_recver():
    global sock
    global recvQueue
    global sockExceptionEvent
    global stopThread

    while not stopThread:
        try:
            data = sock.recv(1048576)
            if not data:
                sockExceptionEvent.set()
                break
            print '#### Received', repr(data)
            recvQueue.put(data)
        except Exception, e:
            sockExceptionEvent.set()
            break
        else:
            pass

def Start():
    t =threading.Thread(target=thread_connector, args=())
    t.start()
    return t

def Run():
    t =Start()
    t.join()


if __name__ == '__main__':
    if len(sys.argv) <4:
        print '%s <server-ip> <server-port> <chid>'%(sys.argv[0])
        print " e.g. python %s 127.0.0.1 8601 ch0 "%(sys.argv[0])
        sys.exit(0)
    serverIp =sys.argv[1]
    serverPort =int(sys.argv[2])
    chid =sys.argv[3]
    Run()

