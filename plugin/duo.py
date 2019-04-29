#!/usr/bin/env python

# @@ duo
# @T type event
# @Q tags: (\"duosecurity\",\"logs\")

import sys
import json
import re

def procln(ev):
    ret = {'valid': False, 'name': 'duo'}
    if 'utctimestamp' not in ev:
        return ret
    ret['timestamp'] = ev['utctimestamp']
    if 'summary' not in ev:
        return ret
    if 'authentication SUCCESS' not in ev['summary']:
        return ret
    if 'details' not in ev:
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
