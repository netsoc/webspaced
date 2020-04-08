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

# Get log in details from the log in screen
@app.route('/api/login', methods=['GET','POST'])
def login():
	attempt = request.get_json()
	email = attempt['details']['email']
	password = attempt['details']['password']

	#TO DO: use Netsoc LDAP to validate user details 

	# state should be 1 if success and user needs to create webspace
	# state should be 2 if success and user has already created webspace
	# state should be 0 if the login was a failure
	return jsonify({'state': 1})

# Return the domains that a webspace is using
@app.route('/api/getDomains', methods=['GET', 'POST'])
def getDomains():

	#TO DO: use API to get current domains

	domains = ['']

	return jsonify({'domains': domains})

# Get submission of domains from user
@app.route('/api/domains', methods=['GET', 'POST'])
def submitDomain():
	attempt = request.get_json()
	domain = attempt['toSubmit']['domain']

	#TO DO: use domain to add domain to the webspace
	
	return jsonify({'result': True, 'domain': domain})

# Return the configurations that a webspace has
@app.route('/api/getConfigs', methods=['GET', 'POST'])
def getConfigs():

	#TO DO: use API to get current configurations that webspace has

	configs = {'HTTP': 8080, 'HTTPS': 8080, 'SSL': True, 'Startup': 1} #temp

	return jsonify(configs)

# Get submission of configurations from user
@app.route('/api/submitConfigs', methods=['GET', 'POST'])
def submitConfigs():
	configs = request.get_json()
	HTTP = configs['configs']['HTTP']
	HTTPS = configs['configs']['HTTPS']
	Startup = configs['configs']['Startup']
	SSL = configs['configs']['SSL']

	#TO DO: use API to configure webspace

	return jsonify({'state': True})