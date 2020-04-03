# pylint: disable=wrong-import-order,wrong-import-position
from gevent import monkey; monkey.patch_all()
from flask_login import login_user, current_user, logout_user, login_required, LoginManager
from flask import Flask, render_template, redirect, url_for
from webspace_ng.forms import LoginForm
import os

app = Flask(__name__)
SECRET_KEY = os.urandom(32)
app.config['SECRET_KEY'] = SECRET_KEY

# Catch-all route for SPA
@app.route('/', defaults={'path': ''})
@app.route('/<path:path>')
def catch_all(path):
    return render_template('index.html')

@app.route("/login", methods=['GET','POST'])
def login():
    return render_template('index.html')
