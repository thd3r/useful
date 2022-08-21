#!/usr/bin/env python3

from __future__ import print_function

from flask import Flask, render_template, send_from_directory

app = Flask(__name__)

@app.route('/')
def index():

    return render_template('index.html')

@app.route('/audio/<path:filename>')
def audio_file(filename):
    
    return send_from_directory("/home/thd3rboy/projects/python/testing/app/audio/", filename)


app.run()