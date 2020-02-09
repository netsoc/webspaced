# pylint: disable=wrong-import-order,wrong-import-position
from gevent import monkey; monkey.patch_all()

from flask import Flask, render_template

app = Flask(__name__)

# Catch-all route for SPA
@app.route('/', defaults={'path': ''})
@app.route('/<path:path>')
def catch_all(path):
    return render_template('index.html')
