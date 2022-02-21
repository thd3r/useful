#!/usr/bin/env python3

import os
import threading
from flask import Flask

os.environ['Flas_ENV'] = 'development'
app = Flask(__name__)
app.debug = False

@app.route('/')
def home():
    return 'Code: 999666'

print("\n * Go to http://127.0.0.1:5000/")
threading.Thread(target=app.run, kwargs={'use_reloader': False}).start()