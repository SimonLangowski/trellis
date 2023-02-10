import json
import sys
import numpy as np
from plot_config import * # plot configuration file

path_establishment = False
title = ["fig1", "fig2", "fig3"]
primary = ["NumServers", "NumUsers", "NumServers"]
secondary  = ["F", "F", "NumUsers"]
fixed = ["NumUsers", "NumServers", "F"]
fixedValue = [1000000, 128, 0.2]
# Increment me if you make the above command line arguments
args = 1

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

data = []
for fileName in sys.argv[args:]:
    data.extend(readAndParse(fileName))
g = groupByNotes(data, path_establishment)

for idx in range(len(title)):
    plotable_data = []

    for p, times in g.items():
        params = dict(p)
        f = params[secondary[idx]]
        servers = params[primary[idx]]
        if params[fixed[idx]] != fixedValue[idx]:
            continue
        times_in_s = np.array([t / 1000000000 for t in times])
        avg = np.mean(times_in_s)
        std = np.std(times_in_s)
        plotable_data.append((servers,f,avg,confidence95(std, len(times))))

    # Sort by numservers
    plotable_data.sort(key=lambda tup: int(tup[0]))
    # Sort by F
    plotable_data.sort(key=lambda tup: float(tup[1]))


    width = 4.5 # default_width
    height = 3.5 # default_height
    def plot_runtime(data):
        ######################## PLOT CODE ########################
        ax = plt.figure(idx).gca()
        ax.yaxis.grid(color=gridcolor, linestyle=linestyle)
        fig = matplotlib.pyplot.gcf()
        fig.set_size_inches(width, height)

        xticks = np.sort(np.unique([d[0] for d in data]))
        curves = np.sort(np.unique([d[1] for d in data]))

        group_number = 0
        i = 0
        while i < len(plotable_data):
            # I do not guarantee all curve will be plotted for all x values
            current_curve = plotable_data[i][1]
            j = i
            while j < len(plotable_data):      
                if current_curve != plotable_data[j][1]:
                    break
                j += 1

            curve_data = plotable_data[i:j]
            print(curve_data)
            # plot
            ax.plot(
                [d[0] for d in curve_data], 
                [d[2] for d in curve_data], 
                marker=markers[0],
                color=colors[group_number],
                lw=linewidth,
                label=curves[group_number]
            )

            plt.fill_between(
                [d[0] for d in curve_data], 
                [d[2]-d[3] for d in curve_data], 
                [d[2]+d[3] for d in curve_data], 
                color=colors[group_number],
                alpha=error_opacity,
            )
            i = j
            group_number += 1

        ax.legend(title=secondary[idx],loc='upper right',  edgecolor='white', framealpha=1, fancybox=False)

        ax.set_xticks(xticks)
        return ax

    print(plotable_data)

    # plot client end-to-end time 
    ax = plot_runtime(plotable_data)
    ax.set_xlabel(primary[idx])
    ax.set_ylabel('Latency (seconds)')
    #ax.set_yscale("log", base=10)
    ax.set_ylim(0, ax.get_ylim()[1] * 1.25) # make y axis 25% bigger
    ax.set_title(title[idx])
    # leg = ax.legend(title='Probes', loc="best", framealpha=1, edgecolor=edgecolor)
    ax.figure.tight_layout()
    ax.figure.savefig(title[idx] + '_latency.png', bbox_inches='tight')
