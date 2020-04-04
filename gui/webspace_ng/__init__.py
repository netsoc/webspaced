# pylint: disable=wrong-import-order,wrong-import-position
from gevent import monkey; monkey.patch_all()
from flask import Flask, jsonify, render_template, redirect, url_for, request
from flask_login import LoginManager, login_user, current_user, logout_user, login_required
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

@app.route('/api/login', methods=['GET','POST'])
def login():
	attempt = request.get_json()
	email = attempt['details']['email']
	password = attempt['details']['password']

	#TO DO: validate the user using Netsoc LDAP

	# state should be 1 if success and user needs to create webspace
	# state should be 2 if success and user has already created webspace
	# state should be 0 if the login was a failure
	return jsonify({'state': 1})