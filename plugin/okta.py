#!/usr/bin/env python

# @@ okta
# @T _type okta
# @T category okta

import sys
import json

def procln(ev):
    ret = {'valid': False}
    if 'details' not in ev:
        return ret
    if 'actors' in ev['details']:
        aval = ev['details']['actors']
        for v in aval:
            if 'ipAddress' in v:
                ret['source_ipv4'] = v['ipAddress']
            if 'login' in v:
                ret['principal'] = v['login']
                ret['valid'] = True
    if 'targets' in ev['details']:
        tval = ev['details']['targets']
        for v in tval:
            if 'login' in v:
                ret['principal'] = v['login']
                ret['valid'] = True
    return ret

ret = {}
ret['results'] = []

inbuf = json.loads(sys.stdin.read())
ret['results'] = [procln(x) for x in inbuf['events']]
sys.stdout.write(json.dumps(ret))

sys.exit(0)
