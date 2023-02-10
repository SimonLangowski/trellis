import json
import sys

fileName = sys.argv[1]
outFileName = "res.csv"
if len(sys.argv) > 2:
    outFileName = sys.argv[2]

def parseJson(s):
    return json.loads(s)

def parseMultiJson(s):
    level = 0
    start = 0
    objs = []
    for pos, c in enumerate(s):
        if c == '{':
            if level == 0:
                start = pos
            level += 1
        elif c == '}':
            level -= 1
            if level == 0:
                objs.append(parseJson(s[start:pos+1]))
    return objs

def readAndParse(fn):
    f = open(fn, "r")
    data = f.read()
    f.close()
    return parseMultiJson(data)

def groupByNotes(d, path):
    grouped = {}
    for x in d:
        if x["Info"].get("pathEstablishment", False) == path :
            n = x["Notes"]
            k = tuple(sorted(n.items()))
            cur = grouped.get(k, [])
            cur.append(x["ServerRoundTime"])
            grouped[k] = cur
    return grouped

output = open(outFileName, "w")
columns = ["F", "NumServers", "NumUsers", "MessageSize"]
output.write(", ".join(columns))
output.write(", Path, Time\n")

def writeOutput(notes, value, path):
    output.write(", ".join([str(notes[c]) for c in columns]))
    output.write(f',{path},{value/1000000000}\n')

def pathEstablishment(d):
    g = groupByNotes(d, True)
    for k,v in g.items():
        s = 0
        # sum times from rounds
        for t in v:
            s += t
        n = dict(k)
        writeOutput(n, s, True)


def broadcast(d):
    g = groupByNotes(d, False)
    for k,v in g.items():
        s = 0
        # average times from rounds
        for t in v:
            s += t
        s /= len(v)
        n = dict(k)
        writeOutput(n, s, False)

data = readAndParse(fileName)
pathEstablishment(data)
broadcast(data)