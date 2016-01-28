# -*- coding: utf-8 -*-

"""
Module implementing DlgChat.
"""

import sys
from PyQt4 import QtGui
from DlgChat import *

from zokket.qt import QtRunloop

if __name__ == "__main__":
    app = QtGui.QApplication(sys.argv)
    
    QtRunloop.set_default(app)
    
    dlg = DlgChat()
    dlg.show()
    sys.exit(app.exec_())
    
