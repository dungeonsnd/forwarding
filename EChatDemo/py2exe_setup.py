#!/bin/python 
# -*- coding:utf-8 -*-

from distutils.core import setup
import py2exe

options = {"py2exe":{"compressed": 1,
                    "optimize": 2,
                    "bundle_files": 1,
                    "dll_excludes": ["MSVCP90.dll","w9xpopen.exe","POWRPROF.dll"],
                    "includes":["zokket","sip"]
            }}


setup(version = "0.1",
    name = "EChat",
    description = "Easy chatting app.",
    windows =[{"script":"EChat.py", "icon_resources":[(1, "ls.ico")]}],
    options=options,
    zipfile="EChat.lib")
