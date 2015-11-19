#!/usr/bin/env python

# @@ google
# @T _type google
# @T category google

import sys
import json

def procln(ev):
    ret = {'valid': False, 'name': 'google'}
    if 'details' not in ev:
        return ret
    if 'events_name' in ev['details']:
        if ev['details']['events_name'] != 'login_success':
            return ret
    if 'sourceipaddress' in ev['details']:
        ret['source_ipv4'] = ev['details']['sourceipaddress']
    if 'actor_email' in ev['details']:
        ret['principal'] = ev['details']['actor_email']
        ret['valid'] = True
    return ret

ret = {}
ret['results'] = []

inbuf = json.loads(sys.stdin.read())
ret['results'] = [procln(x) for x in inbuf['events']]
sys.stdout.write(json.dumps(ret))

sys.exit(0)
