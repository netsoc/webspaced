from functools import wraps
import sys
import os
import signal
import termios
import tty
import socket
import select
import shutil
import urllib
import getpass
import traceback

import requests
import requests_unixsocket
from humanfriendly import format_size, format_timespan
from eventfd import EventFD

requests_unixsocket.monkeypatch()

CONSOLE_ESCAPE = b'\x1d'
CONSOLE_ESCAPE_QUIT = b'q'

class WebspaceError(Exception):
    pass

def ask(question, default="yes"):
    """Ask a yes/no question via input() and return their answer.

    "question" is a string that is presented to the user.
    "default" is the presumed answer if the user just hits <Enter>.
        It must be "yes" (the default), "no" or None (meaning
        an answer is required of the user).

    The "answer" return value is True for "yes" or False for "no".
    """
    valid = {"yes": True, "y": True, "ye": True,
             "no": False, "n": False}
    if default is None:
        prompt = " [y/n] "
    elif default == "yes":
        prompt = " [Y/n] "
    elif default == "no":
        prompt = " [y/N] "
    else:
        raise ValueError("invalid default answer: '{}'".format(default))

    while True:
        print(question + prompt, end='')
        choice = input().lower()
        if default is not None and choice == '':
            return valid[default]
        elif choice in valid:
            return valid[choice]
        else:
            print("Please respond with 'yes' or 'no' "
                             "(or 'y' or 'n').")

class process:
    def __init__(self, message, done=' done.'):
        self.message = message
        self.done = done
    def __enter__(self):
        print(self.message, end='')
        sys.stdout.flush()
        return self
    def __exit__(self, ex_type, e_value, trace):
        if not e_value:
            print(self.done)
        else:
            print()

class Client:
    def __init__(self, sock, user=None):
        self.base = f'http+unix://{urllib.parse.quote(sock, safe="")}'
        self.user = user

    def req(self, method, path, body=None, plain=False):
        headers = {}
        if self.user:
            headers['X-Webspace-User'] = self.user

        res = requests.request(method, f'{self.base}{path}', headers=headers, json=body)
        if res.status_code >= 400:
            try:
                raise WebspaceError(res.json()['message'])
            except (ValueError, KeyError):
                res.raise_for_status()

        if res.status_code != 204:
            if plain:
                return res.text
            return res.json()

def cmd(f):
    @wraps(f)
    def wrapper(args):
        user = args.user if 'user' in args else None
        client = Client(args.socket_path, user=user)
        try:
            return f(client, args)
        except Exception as ex:
            print(f'Error: {ex}', file=sys.stderr)
    return wrapper

def find_image(client, id_):
    image_list = client.req('GET', '/v1/images')
    # First try to find it by an alias
    for i in image_list:
        for a in i['aliases']:
            if a['name'] == id_:
                return i

    # Otherwise by fingerprint
    for i in image_list:
        if i['fingerprint'] == id_:
            return i

    return None

@cmd
def images(client, _args):
    image_list = client.req('GET', '/v1/images')
    print('Available images: ')
    for image in image_list:
        print(' - Fingerprint: {}'.format(image['fingerprint']))
        if image['aliases']:
            aliases = map(lambda a: a['name'], image['aliases'])
            print('   Aliases: {}'.format(', '.join(aliases)))
        if 'description' in image['properties']:
            print('   Description: {}'.format(image['properties']['description']))
        print('   Size: {}'.format(format_size(image['size'], binary=True)))

@cmd
def init(client, args):
    image = find_image(client, args.image)
    if image is None:
        raise WebspaceError(f'"{args.image}" is not a valid image alias / fingerprint')
    body = {
        'image': image['fingerprint']
    }

    if not args.no_password:
        body['password'] = getpass.getpass('New root password: ')
        if getpass.getpass('Confirm: ') != body['password']:
            raise WebspaceError("Passwords don't match!")

    if 'ssh_key' in args:
        body['sshKey'] = args.ssh_key

    with process('Creating your webspace...', done=' success!'):
        client.req('POST', '/v1/webspace', body)

@cmd
def status(client, _args):
    info = client.req('GET', '/v1/webspace/state')
    print(f'Webspace is {"" if info["running"] else "not "}running')

    if info['usage']['disks']:
        print('Disks:')
        for name, usage in info['usage']['disks'].items():
            print(f' - {name}: Used {format_size(usage, binary=True)}')

    if not info["running"]:
        return

    print(f'CPU time: {format_timespan(info["usage"]["cpu"] / 1000 / 1000 / 1000)}')

    print(f'Memory usage: {format_size(info["usage"]["memory"], binary=True)}')

    print(f'Running processes: {info["usage"]["processes"]}')

    if info['networkInterfaces']:
        print('Network interfaces:')
        for name, data in info['networkInterfaces'].items():
            print(f' - {name} ({data["mac"]}):')
            print('   Sent/received: {}/{}'.format(
                format_size(data['counters']['bytesSent'], binary=True),
                format_size(data['counters']['bytesReceived'], binary=True)))
            for addr in data['addresses']:
                print('   IPv{} address: {}/{}'.format('6' if addr['family'] == 'inet6' else '4',
                    addr['address'], addr['netmask']))

@cmd
def log(client, _args):
    print(client.req('GET', '/v1/webspace/console', plain=True))

#def _console(client, command=None, environment={}):
#    t_width, t_height = shutil.get_terminal_size()
#    if not command:
#        print('Attaching to console...')
#        sock_path = client.console(t_width, t_height)
#    else:
#        sid, sock_path = client.exec(command, t_width, t_height, environment)
#
#    def notify_resize(_signum, _frame):
#        t_width, t_height = shutil.get_terminal_size()
#        if not command:
#            client.console_resize(t_width, t_height)
#        else:
#            client.exec_resize(sid, t_width, t_height)
#    # SIGWINCH is sent when the terminal is resized
#    signal.signal(signal.SIGWINCH, notify_resize)
#
#    # Establish the terminal pipe connection
#    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
#    sock.connect(sock_path)
#
#    stdin = sys.stdin.fileno()
#    old = termios.tcgetattr(stdin)
#    tty.setraw(stdin, when=termios.TCSANOW)
#
#    should_quit = EventFD()
#    def trigger_quit(_signum, _frame):
#        should_quit.set()
#    signal.signal(signal.SIGINT, trigger_quit)
#    signal.signal(signal.SIGTERM, trigger_quit)
#    if not command:
#        print('Attached, hit ^] (Ctrl+]) and then q to disconnect', end='\r\n')
#
#    try:
#        escape_read = False
#        while True:
#            r, _, _ = select.select([should_quit, sys.stdin, sock], [], [])
#            if should_quit in r:
#                break
#            if sys.stdin in r:
#                data = os.read(stdin, 1)
#                if not command:
#                    if escape_read:
#                        if data == CONSOLE_ESCAPE_QUIT:
#                            # The user wants to quit
#                            break
#
#                        # They don't want to quit, lets send the escape key along with their data
#                        sock.sendall(CONSOLE_ESCAPE + data)
#                        escape_read = False
#                    elif data == CONSOLE_ESCAPE:
#                        escape_read = True
#                    else:
#                        sock.sendall(data)
#                else:
#                    sock.sendall(data)
#            if sock in r:
#                data = sock.recv(4096)
#                if not data:
#                    break
#
#                sys.stdout.buffer.write(data)
#                sys.stdout.flush()
#    finally:
#        # Restore the terminal to its original state
#        termios.tcsetattr(stdin, termios.TCSANOW, old)
#        sock.close()
#
#@cmd
#def exec(client, args):
#    _console(client, command=[args.command] + args.args)
#@cmd
#def console(client, _args):
#    _console(client)

@cmd
def boot(client, _args):
    with process('Starting your webspace...'):
        client.req('POST', '/v1/webspace/state')

@cmd
def shutdown(client, _args):
    with process('Shutting your webspace down...'):
        client.req('DELETE', '/v1/webspace/state')

@cmd
def reboot(client, _args):
    with process('Rebooting your webspace...'):
        client.req('PUT', '/v1/webspace/state')

@cmd
def delete(client, _args):
    if not ask('Are you sure?', default='no'):
        return

    with process('Deleting your webspace...'):
        client.req('DELETE', '/v1/webspace')

@cmd
def config_show(client, _args):
    config = client.req('GET', '/v1/webspace/config')
    print('Webspace configuration:')

    print(f'Startup delay: {format_timespan(config["startupDelay"])}')
    print(f'HTTP port: {config["httpPort"]}')

    if config['httpsPort'] == 0:
        print('SSL termination is enabled')
    else:
        print(f'SSL termination is disabled - HTTPS port: {config["httpsPort"]}')
@cmd
def config_set(client, args):
    if args.option in ('httpPort', 'httpsPort'):
        args.value = int(args.value)
    elif args.option == 'startupDelay':
        args.value = float(args.value)

    client.req('PATCH', '/v1/webspace/config', {
        args.option: args.value
    })

@cmd
def domains_show(client, _args):
    domains = client.req('GET', '/v1/webspace/domains')
    print('Webspace domains:')
    for domain in domains:
        print(' - {}'.format(domain))
@cmd
def domains_add(client, args):
    with process('Verifying and adding domain...'):
        client.req('POST', f'/v1/webspace/domains/{args.domain}')
@cmd
def domains_remove(client, args):
    client.req('DELETE', f'/v1/webspace/domains/{args.domain}')

@cmd
def ports_show(client, _args):
    ports = client.req('GET', '/v1/webspace/ports')
    print('Webspace port forwards:')
    for e, i in ports.items():
        print(f' - {e} -> {i}')
@cmd
def ports_add(client, args):
    if args.eport == 0:
        args.eport = client.req('POST', f'/v1/webspace/ports/{args.iport}')['ePort']
    else:
        client.req('POST', f'/v1/webspace/ports/{args.eport}/{args.iport}', plain=True)
    print(f'Port {args.iport} in your webspace is now accessible externally via port {args.eport}')
@cmd
def ports_remove(client, args):
    for port in args.port:
        client.req('DELETE', f'/v1/webspace/ports/{port}')

#@cmd
#def login(client, _args):
#    config = client.get_config()
#    if 'name' in config:
#        user = config['name']
#    else:
#        user = 'root'
#        print('Warning: `name` config option is not set - defaulting to `root`')
#        print('(Use `{} config set user <username>` to set this option)'.format(sys.argv[0]))
#
#    # `script` is a workaround for LXD's lack of pts allocation with `exec`
#    env = {'TERM': os.environ.get('TERM', 'vt100')}
#    _console(client, ['script', '-q', '-c', 'su - {}'.format(user), '/dev/null'], environment=env)
