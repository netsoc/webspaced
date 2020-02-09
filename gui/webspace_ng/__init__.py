# pylint: disable=wrong-import-order,wrong-import-position
from gevent import monkey; monkey.patch_all()

from flask import Flask

app = Flask(__name__)

@app.route('/')
def index():
    return 'hello, world!'
