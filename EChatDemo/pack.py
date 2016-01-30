# -*- coding: utf-8 -*-

import hashlib
import math
import struct
import base64
import json
import zlib

import binascii

from Crypto.Cipher import AES
from Crypto import Random

salt ='__E3S$hH%&*KL:"II<UG=_!@fc9}021jFJ|KDI.si81&^&%%^*(del?%)))+__'
fingerprint_len =4
iv_len =16
randomiv_len =4

print_log =False

# 输入密码，输出其hash值的前两个字节的16进制表示.
def fingerprintSimple(input_str):
    return binascii.hexlify(hashlib.sha256(input_str).digest()[0:2])

def hash(input):
    return hashlib.sha256(input).digest()
    
def fingerprint(input):
	return struct.pack('!i',zlib.adler32(input))
    
def pack(pwd, dict_input):
    try:
        if print_log:
            print 'pack pwd=', pwd
            print 'pack dict_input=', dict_input
            
        input =json.dumps(dict_input)
        
        l =len(input)
        output =input.ljust(int(math.ceil(l/16.0)*16),  ' ')
        
        rndfile = Random.new()
        randomiv =rndfile.read(randomiv_len)
        iv =hash(randomiv)[0:iv_len]
        if print_log:
            print 'pack iv=', repr(iv)
        
        key =hash(salt+pwd)
        encryptor =AES.new(key, AES.MODE_CBC,  iv)
        encrypted_str = encryptor.encrypt(output)
        
        output =randomiv+encrypted_str
        
        fp =fingerprint(output)
        
        # body_len + fp + randomiv + encrypted_msg + padding
        body_len =struct.pack('!i', l)
        output =body_len+fp+output
        
        if print_log:
            print 'pack body_len=', l
            print 'pack randomiv=', repr(randomiv)
            print 'pack fingerprint=', repr(fp)
            print 'pack encrypted_str=%s, len=%d'% (repr(encrypted_str), len(encrypted_str))

        
        output =base64.b64encode(output)
        
        if print_log:
            print 'pack result:%s, len=%d' %(output, len(output))
        output =output+'\r\n'
        return output
    except:
        return ''
def unpack(pwd, input_str_utf8):
    try:
        if input_str_utf8[-2: ]=='\r\n':
            input =input_str_utf8[0: len(input_str_utf8)-2]
        else :
            input =input_str_utf8
        
        if print_log:
            print 'unpack input:%s, len=%d' %(input, len(input))
        input =base64.b64decode(input)

        # body_len + fp + randomiv + encrypted_msg + padding
        l,  =struct.unpack('!i', input[0:4])
        if print_log:
            print 'unpack body_len=', l
        input =input[4:]
        
        if print_log:
            print 'unpack input fingerprint=', repr(input[0:fingerprint_len])
            print 'unpack cal fingerprint=', repr(fingerprint(input[fingerprint_len:]))
        if fingerprint(input[fingerprint_len:])!=input[0:fingerprint_len]:
            return {}
        input =input[fingerprint_len:]
        
        randomiv =input[0:randomiv_len]
        iv =hash(randomiv)[0:iv_len]
        input =input[randomiv_len:]
        if print_log:
            print 'unpack randomiv=', repr(randomiv)
            print 'unpack iv=', repr(iv)
        
        key =hash(salt+pwd)
        decryptor =AES.new(key, AES.MODE_CBC,  iv)
        output = decryptor.decrypt(input)
        output =output[0:l]
        
        if print_log:
            print 'unpack, json.loads data:', output
        d =json.loads(output)  
        if print_log:
            print 'unpack result:', d
        return d
    except:
        return {}

if __name__=='__main__':
    d ={'k':u'大神好'}
    print 'pack input=',d
    enc =pack('qwert',d)
    print 'pack result=',enc
    d =unpack('qwert',enc)
    print 'unpack result=',d
	
