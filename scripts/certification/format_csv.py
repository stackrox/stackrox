import heapq
import sys

def usage():
    print('usage: python3 format_csv.py <input file>.csv <output file>.csv')
    sys.exit(1)

if len(sys.argv) != 3:
    usage()

input_file = sys.argv[1]
output_file = sys.argv[2]

if input_file[-4:] != '.csv':
    usage()
if output_file[-4:] != '.csv':
    usage()

final = {}
cve_heap = []

with open(input_file, 'r') as input:
    for line in input.readlines():
        parts = line.split(",")
        name = parts[0][1:-1]
        version = parts[1][1:-1]
        cve = parts[2][1:-1]
        cvss = parts[3]
        fixed_by = parts[4]
        if len(fixed_by) != "":
            fixed_by = fixed_by[1:-1]
        link = parts[5][1:-1]
        severity = parts[6][1:-2]
        if cve not in final:
            final[cve] = {"package_heap": [], "data": {}}
            heapq.heappush(cve_heap, cve)
        if name not in final[cve]["data"]:
            heapq.heappush(final[cve]["package_heap"], name)
        final[cve]["data"][name] = {"version": version, "severity": severity, "cvss": cvss, "fixedBy": fixed_by, "link": link}

with open(output_file, 'w') as output:
    output.write("vuln,package,version,fix version,severity,score,link\n")
    while len(cve_heap) > 0:
        cve = heapq.heappop(cve_heap)
        while len(final[cve]["package_heap"]) > 0:
            package = heapq.heappop(final[cve]["package_heap"])
            data = final[cve]["data"][package]
            output.write("{},{},{},{},{},{},{}\n".format(cve, package, data["version"], data["fixedBy"], data["severity"], data["cvss"], data["link"]))
