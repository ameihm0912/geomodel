#!/usr/bin/env python

# @@ bmo
# @T _type event
# @T category syslog
# @T details.program apache
# @Q summary: \\[audit\\]
# @Q summary: login

import sys
import json
import re

def procln(ev):
    ret = {'valid': False, 'name': 'bmo'}
    if 'utctimestamp' not in ev:
        return ret
    ret['timestamp'] = ev['utctimestamp']
    if 'summary' not in ev:
        return ret
    mtch = re.match('^.*successful login of (\S+) from (\S+).*', ev['summary'])
    if mtch == None:
        return ret
    ret['principal'] = mtch.group(1)
    ret['source_ipv4'] = mtch.group(2)
    ret['valid'] = True
    return ret

ret = {}
ret['results'] = []

inbuf = json.loads(sys.stdin.read())
ret['results'] = [procln(x) for x in inbuf['events']]
sys.stdout.write(json.dumps(ret))

sys.exit(0)
