#!/usr/bin/env python

# @@ auth0
# @T _type event
# @Q tags: auth0

import sys
import json

SUCCESS_LOGIN_TEXTS = [
    "Success Login",
    "Success Silent Auth"
]


def procln(ev):
    ret = {'valid': False, 'name': 'auth0'}
    if 'utctimestamp' not in ev:
        return ret
    ret['timestamp'] = ev['utctimestamp']

    if 'details' not in ev:
        return ret
    if 'type' not in ev['details'] or ev['details']['type'] not in SUCCESS_LOGIN_TEXTS:
        return ret

    if 'sourceipaddress' not in ev['details'] or ev['details']['sourceipaddress'] == '0.0.0.0':
        return ret
    if 'username' not in ev['details']:
        return ret

    ret['principal'] = ev['details']['username']
    ret['source_ipv4'] = ev['details']['sourceipaddress']
    ret['valid'] = True
    return ret


ret = {}
ret['results'] = []

inbuf = json.loads(sys.stdin.read())
ret['results'] = [procln(x) for x in inbuf['events']]
sys.stdout.write(json.dumps(ret))

sys.exit(0)
