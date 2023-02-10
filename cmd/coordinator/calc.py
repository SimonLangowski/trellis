import json
import sys
import csv

headers = ["NumServers", "NumUsers", "F", "path", "NumLayers", "GroupSize", "NumGroups", "BinSize", "NumClientServers", "ctime", "stime"]

def main():
    file = sys.argv[1]
    f = open(file, "r")
    contents = f.read()
    objects = contents.split("}{")
    results = []
    for o in objects:
        if not o.startswith("{"):
            o = "{" + o
        if not o.endswith("}"):
            o = o + "}"
        j = json.loads(o)
        results.append(j)
    pathTimes = {}
    broadcastTimes = {}
    for r in results:
        pathEstablishment = r["Info"].get("pathEstablishment", False)
        ctime = r["ClientAndServerTokenTime"]
        stime = r["ServerRoundTime"]
        k = tuple(sorted((r["Notes"].items())))
        if pathEstablishment :
            acc = pathTimes.get(k, {"ctime": 0, "stime": 0, "path": True})
            acc["ctime"] += ctime
            acc["stime"] += stime
            pathTimes[k] = acc
        else:
            acc = broadcastTimes.get(k, {"ctime": 0, "stime": 0, "obs": 0, "path": False})
            acc["ctime"] += ctime
            acc["stime"] += stime
            acc["obs"] += 1
            broadcastTimes[k] = acc
    for r in broadcastTimes.values():
        r["ctime"] /= r["obs"]
        r["stime"] /= r["obs"]
    
    f = open(file + ".csv", "w")
    w = csv.writer(f)
    w.writerow(headers)
    for (k, v) in pathTimes.items():
        d = dict(k)
        row = []
        for h in headers:
            val = d.get(h)
            if val == None:
                val = v.get(h)
            row.append(val)
        w.writerow(row)
    for (k, v) in broadcastTimes.items():
        d = dict(k)
        row = []
        for h in headers:
            val = d.get(h)
            if val == None:
                val = v.get(h)
            row.append(val)
        w.writerow(row)
    f.flush()

if __name__ == "__main__":
    main()