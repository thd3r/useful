#!/usr/bin/env python3

import sys
import os
import time
import requests
import threading
from flask import Flask, request, render_template
from pyngrok import ngrok

os.environ["FLASK_ENV"] = "development"
app = Flask(__name__)
app.debug = False
port = 5000

public_url = ngrok.connect(port).public_url
final_url = public_url[:4] + public_url[4:]

print(" * ngrok tunnel -> \033[93m{}\033[0m".format(final_url))
app.config['BASE_URL'] = public_url

@app.route('/', methods=['GET', 'POST'])
def home():
    if request.method == 'GET':
        req = requests.get('http://localhost:4040/api/requests/http').json()
        user_agent = req['requests'][0]['request']['headers']['User-Agent'][0]
        ip_address = req['requests'][0]['request']['headers']['X-Forwarded-For'][0]
        now = time.strftime('%m/%d/%Y %H:%M:%S %p')
        print(f'\n * Time: {now}')
        print(f' * User-Agent: {user_agent}')
        print(f' * Ip address: {ip_address}\n')

    else:
        pass
    return render_template('index.html')

threading.Thread(target=app.run, kwargs={'use_reloader': False}).start()
