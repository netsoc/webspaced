import os
import pwd
import grp
import argparse

from .commands import *

def main():
    parser = argparse.ArgumentParser(formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument('-c', '--socket', dest='socket_path',
                        help="Path to the daemon's Unix socket",
                        default='/run/webspaced/server.sock')
    parser.add_argument('-u', '--user', help='User to perform operations as (only works if you are webspace admin)')

    subparsers = parser.add_subparsers(required=True, dest='command')

    p_images = subparsers.add_parser('images', help='List available images')
    p_images.set_defaults(func=images)

    p_init = subparsers.add_parser('init', help='Create your webspace',
                                   formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    p_init.add_argument('--no-password', help='Disable root password', action='store_true')
    p_init.add_argument('-k', '--ssh-key', help='SSH public key to install in webspace (creates port forward)')
    p_init.add_argument('image',
                        help='Image alias / fingerprint to create your webspace from')
    p_init.set_defaults(func=init)

    p_status = subparsers.add_parser('status', help='Show the status of your webspace')
    p_status.set_defaults(func=status)

    #p_log = subparsers.add_parser('log', help="Retrieve your webspace's system log")
    #p_log.set_defaults(func=log)

    #p_exec = subparsers.add_parser('exec', help='Run a command in your webspace')
    #p_exec.add_argument('command', help='Command to run')
    #p_exec.add_argument('args', nargs=argparse.REMAINDER, help='Command arguments')
    #p_exec.set_defaults(func=exec)

    #p_console = subparsers.add_parser('console', help="Attach to your webspace's console")
    #p_console.set_defaults(func=console)

    #p_exec = subparsers.add_parser('login', help='Shortcut to open a shell in your webspace \
    #    (logging in with a configured user)')
    #p_exec.set_defaults(func=login)

    p_boot = subparsers.add_parser('boot', help='Start your webspace')
    p_boot.set_defaults(func=boot)

    p_shutdown = subparsers.add_parser('shutdown', help='Shutdown your webspace')
    p_shutdown.set_defaults(func=shutdown)

    p_reboot = subparsers.add_parser('reboot', help='Reboot your webspace')
    p_reboot.set_defaults(func=reboot)

    p_delete = subparsers.add_parser('delete', help='Delete your webspace')
    p_delete.set_defaults(func=delete)

    p_config = subparsers.add_parser('config', help="Change your webspace's options")
    p_config.set_defaults(func=config_show)
    cfg_sub = p_config.add_subparsers(dest='cfg_command')

    cfg_show = cfg_sub.add_parser('show', help='Show webspace configuration')
    cfg_show.set_defaults(function=config_show)

    cfg_set = cfg_sub.add_parser('set', help='Set a config option')
    cfg_set.add_argument('option', help='Name of option to set', choices=['startupDelay', 'httpPort', 'httpsPort'])
    cfg_set.add_argument('value', help='Value of option to set')
    cfg_set.set_defaults(func=config_set)

    p_dns = subparsers.add_parser('domains', help='Configure custom domains')
    p_dns.set_defaults(func=domains_show)
    dns_sub = p_dns.add_subparsers(dest='dns_command')

    dns_show = dns_sub.add_parser('show', help='Show configured custom domains')
    dns_show.set_defaults(function=domains_show)

    dns_add = dns_sub.add_parser('add', help='Add a custom domain')
    dns_add.add_argument('domain', help='Domain to add')
    dns_add.set_defaults(func=domains_add)

    dns_remove = dns_sub.add_parser('remove', help='Remove a custom domain')
    dns_remove.add_argument('domain', help='Domain to remove')
    dns_remove.set_defaults(func=domains_remove)

    #p_pf = subparsers.add_parser('ports', help='Configure forwarded ports')
    #p_pf.set_defaults(func=ports_show)
    #pf_sub = p_pf.add_subparsers(dest='ports_command')

    #pf_show = pf_sub.add_parser('show', help='Show forwarded ports')
    #pf_show.set_defaults(function=ports_show)

    #pf_add = pf_sub.add_parser('add', help='Forward a port')
    #pf_add.add_argument('iport', help='Internal port', type=int)
    #pf_add.add_argument('-p', '--external-port', dest='eport', help='External port (0 means random)', type=int, default=0)
    #pf_add.set_defaults(func=ports_add)

    #pf_remove = pf_sub.add_parser('remove', help='Remove a forwarded port')
    #pf_remove.add_argument('iport', help='Internal port', type=int)
    #pf_remove.set_defaults(func=ports_remove)

    args = parser.parse_args()
    args.func(args)
