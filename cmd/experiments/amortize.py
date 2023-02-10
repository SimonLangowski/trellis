
columns = ['numservers','numusers','f','messagesize','numlayers','binsize','bandwidth','latency','round','pathestablishment','roundtime']

def readFile(fn, pathOnly=False):
    f = open(fn, 'r')
    data = {}
    for l in f.readlines():
        if l[0].isalpha() or l[0].isspace():
            continue
        cols = l.split(',')
        for idx, c in enumerate(cols):
            cols[idx] = c.strip()
        params = ','.join(cols[:7])
        round = int(cols[8])
        if round == 0:
            continue
        path = cols[9] == 'true'
        if path != pathOnly:
            continue
        time = int(cols[10])
        current = data.get(params, [])
        current.append(time)
        data[params] = current
    return data

path = readFile("plot5.csv", True)
totals = {}
for params, values in path.items():
    totals[params] = sum(values)

lightning = readFile("plot1.csv", False)
means = {}
for params, values in lightning.items():
    means[params] = sum(values)/len(values)

size = 1000000
forSize = {}
for params, mean in means.items():
    cols = params.split(',')
    newParams = cols[0] + ',' + cols[1] + ',' + cols[2] + ',' + cols[3]
    forSize[newParams] = size / int(cols[3]) * mean

totalForSize = {}
for params, cost in forSize.items():
    pathCost = 0
    cols = params.split(',')
    sharedParams = cols[0] + ',' + cols[1] + ',' + cols[2] 
    for p in totals:
        if p.startswith(sharedParams):
            pathCost = totals[p]
    if pathCost == 0:
        print("Error path not found")
    else:
        totalForSize[params] = cost + pathCost

for params, cost in totalForSize.items():
    print(params + "," + str(cost))
    