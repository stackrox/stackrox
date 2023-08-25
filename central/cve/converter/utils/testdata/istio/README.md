Fixture is generated in the following way:
1. took one record from CVE list downloaded from NVD feed: https://nvd.nist.gov/vuln/data-feeds#JSON_FEED
	-> save single JSON record in single-cve.json
2. generate all paths: https://www.convertjson.com/json-path-list.htm
	-> save result in all-paths.txt
3. generate versions without each path with the following command:
	cat all-paths.txt | awk '{print "jq '"'"'del(" $0 ")'"'"' -c single-cve.json >> cve-list-panic.json"}' | xargs -0 bash -c
4. add null record and array wrapper manually