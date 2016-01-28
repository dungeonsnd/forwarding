from setuptools import setup
APP = ['EChat.py']
DATA_FILES = ['DlgChat.ui']
OPTIONS = {'argv_emulation': False, 'iconfile':'ls.ico', 'includes':['sip', 'PyQt4', 'PyQt4.QtGui', 'PyQt4.QtCore', 'PyQt4.uic', 'sitecustomize']}
setup(
    app=APP,
    data_files=DATA_FILES,
    options={'py2app': OPTIONS},
    setup_requires=['py2app'],
)
