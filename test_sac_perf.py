import datetime
import json
import requests
import sys
import time
import urllib3

##### CONSTANTS #####

HOST = 'https://localhost:8042'
#HOST = 'https://localhost:9000'
#HOST = 'https://34.134.207.6'

#PASSWORD_FILE_PATH = 'deploy/k8s/central-deploy/password'
PASSWORD_FILE_PATH = 'yann-postgres-sac-performnce-05.pwd'
#PASSWORD_FILE_PATH = 'yann-postgres-sac-performnce-06.pwd'
#PASSWORD_FILE_PATH = 'sac_perf_test_pwd'

GRAPHQL_ENDPOINT             = '/api/graphql'
NAMESPACE_ENDPOINT           = '/v1/namespaces'
ROLE_ENDPOINT                = '/v1/roles'
SIMPLE_ACCESS_SCOPE_ENDPOINT = '/v1/simpleaccessscopes'
TOKEN_GENERATOR_ENDPOINT     = '/v1/apitokens/generate'

ACCESS_SCOPE_PREFIX = 'io.stackrox.authz.accessscope.'
PERMISSION_SET_PREFIX = 'io.stackrox.authz.permissionset.'

EMPTY_QUERY = ''

# MAP KEYS
K_ACCESSSCOPEID       = 'accessScopeId'
K_ACCESSSCOPENAME     = 'accessScopeName'
K_CLUSTERNAME         = 'clusterName'
K_DESCRIPTION         = 'description'
K_ID                  = 'id'
K_INCLUDED_NAMESPACES = 'included_namespaces'
K_METADATA            = 'metadata'
K_NAME                = 'name'
K_NAMESPACENAME       = 'namespaceName'
K_NAMESPACES          = 'namespaces'
K_PERMISSIONSETID     = 'permissionSetId'
K_RESOURCETOACCESS    = 'resourceToAccess'
K_ROLES               = 'roles'
K_RULES               = 'rules'
K_TOKEN               = 'token'

#TEST_RUN_COUNTS = 1
TEST_RUN_COUNTS = 10
##TEST_RUN_COUNTS = 100
QUERY_TIMEOUT = 1200

GRAPHQL_WIDGET_QUERIES = {
  'summary_counts':            '{"operationName":"summary_counts","variables":{},"query":"query summary_counts {\\n  clusterCount\\n  nodeCount\\n  violationCount\\n  deploymentCount\\n  imageCount\\n  secretCount\\n}\\n"}',
  'getAllNamespacesByCluster': '{"operationName":"getAllNamespacesByCluster","variables":{"query":""},"query":"query getAllNamespacesByCluster($query: String) {\\n  clusters(query: $query) {\\n    id\\n    name\\n    namespaces {\\n      metadata {\\n        id\\n        name\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
  'mostRecentAlerts':          '{"operationName":"mostRecentAlerts","variables":{"query":"Severity:CRITICAL_SEVERITY"},"query":"query mostRecentAlerts($query: String) {\\n  alerts: violations(\\n    query: $query\\n    pagination: {limit: 3, sortOption: {field: \\"Violation Time\\", reversed: true}}\\n  ) {\\n    id\\n    time\\n    deployment {\\n      clusterName\\n      namespace\\n      name\\n      __typename\\n    }\\n    policy {\\n      name\\n      severity\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
  'getImagesDashboard':        '{"operationName":"getImagesDashboard","variables":{"query":""},"query":"query getImages($query: String) {\\n  images(\\n    query: $query\\n    pagination: {limit: 6, sortOption: {field: \\"Image Risk Priority\\", reversed: false}}\\n  ) {\\n    id\\n    name {\\n      remote\\n      fullName\\n      __typename\\n    }\\n    priority\\n    imageVulnerabilityCounter {\\n      important {\\n        total\\n        fixable\\n        __typename\\n      }\\n      critical {\\n        total\\n        fixable\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
  'agingImagesQuery':          '{"operationName":"agingImagesQuery","variables":{"query0":"Image Created Time:30d-90d","query1":"Image Created Time:90d-180d","query2":"Image Created Time:180d-365d","query3":"Image Created Time:>365d"},"query":"query agingImagesQuery($query0: String, $query1: String, $query2: String, $query3: String) {\\n  timeRange0: imageCount(query: $query0)\\n  timeRange1: imageCount(query: $query1)\\n  timeRange2: imageCount(query: $query2)\\n  timeRange3: imageCount(query: $query3)\\n}\\n"}',
  'getAggregatedResults':      '{"operationName":"getAggregatedResults","variables":{"groupBy":["STANDARD"],"where":"Cluster:*"},"query":"query getAggregatedResults($groupBy: [ComplianceAggregation_Scope!], $where: String) {\\n  controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {\\n    results {\\n      aggregationKeys {\\n        id\\n        scope\\n        __typename\\n      }\\n      numFailing\\n      numPassing\\n      numSkipped\\n      unit\\n      __typename\\n    }\\n    __typename\\n  }\\n  complianceStandards: complianceStandards {\\n    id\\n    name\\n    __typename\\n  }\\n}\\n"}',

  'cvesCount':                                 '{"operationName":"cvesCount","variables":{"query":"Fixable:true"},"query":"query cvesCount($query: String) {\\n  imageVulnerabilityCount\\n  fixableImageVulnerabilityCount: imageVulnerabilityCount(query: $query)\\n  nodeVulnerabilityCount\\n  fixableNodeVulnerabilityCount: nodeVulnerabilityCount(query: $query)\\n  clusterVulnerabilityCount\\n  fixableClusterVulnerabilityCount: clusterVulnerabilityCount(query: $query)\\n}\\n"}',
  'policiesCount':                             '{"operationName":"policiesCount","variables":{"query":"Category:Vulnerability Management"},"query":"query policiesCount($query: String) {\\n  policies(query: $query) {\\n    id\\n    alertCount\\n    __typename\\n  }\\n}\\n"}',
  'getNodes':                                  '{"operationName":"getNodes","variables":{"query":""},"query":"query getNodes($query: String) {\\n  nodeCount(query: $query)\\n}\\n"}',
  'getImages':                                 '{"operationName":"getImages","variables":{"query":""},"query":"query getImages($query: String) {\\n  imageCount(query: $query)\\n}\\n"}',
  'topRiskyDeployments':                       '{"operationName":"topRiskyDeployments","variables":{"query":"","vulnQuery":"","entityPagination":{"offset":0,"limit":25,"sortOption":{"field":"Deployment Risk Priority","reversed":false}},"vulnPagination":{"offset":0,"limit":50,"sortOption":{"field":"CVSS","reversed":true}}},"query":"query topRiskyDeployments($query: String, $vulnQuery: String, $entityPagination: Pagination, $vulnPagination: Pagination) {\\n  results: deployments(query: $query, pagination: $entityPagination) {\\n    id\\n    name\\n    clusterName\\n    namespaceName: namespace\\n    priority\\n    plottedVulns: plottedImageVulnerabilities(query: $vulnQuery) {\\n      basicVulnCounter: basicImageVulnerabilityCounter {\\n        all {\\n          total\\n          fixable\\n          __typename\\n        }\\n        __typename\\n      }\\n      vulns: imageVulnerabilities(pagination: $vulnPagination) {\\n        ...vulnFields\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n\\nfragment vulnFields on ImageVulnerability {\\n  id\\n  cve\\n  cvss\\n  severity\\n  __typename\\n}\\n"}',
  'topRiskiestImageVulns':                     '{"operationName":"topRiskiestImageVulns","variables":{"query":"","pagination":{"offset":0,"limit":8,"sortOption":{"field":"Image Risk Priority","reversed":false}}},"query":"query topRiskiestImageVulns($query: String, $pagination: Pagination) {\\n  results: images(query: $query, pagination: $pagination) {\\n    id\\n    name {\\n      fullName\\n      __typename\\n    }\\n    vulnCounter: imageVulnerabilityCounter {\\n      all {\\n        total\\n        fixable\\n        __typename\\n      }\\n      low {\\n        total\\n        fixable\\n        __typename\\n      }\\n      moderate {\\n        total\\n        fixable\\n        __typename\\n      }\\n      important {\\n        total\\n        fixable\\n        __typename\\n      }\\n      critical {\\n        total\\n        fixable\\n        __typename\\n      }\\n      __typename\\n    }\\n    priority\\n    scan {\\n      scanTime\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
  'frequentlyViolatedPolicies':                '{"operationName":"frequentlyViolatedPolicies","variables":{"query":"+\\n            Category:Vulnerability Management"},"query":"query frequentlyViolatedPolicies($query: String) {\\n  results: policies(query: $query) {\\n    id\\n    name\\n    enforcementActions\\n    severity\\n    alertCount\\n    categories\\n    description\\n    latestViolation\\n    __typename\\n  }\\n}\\n"}',
  'recentlyDetectedImageVulnerabilities':      '{"operationName":"recentlyDetectedImageVulnerabilities","variables":{"query":"CVE Type:IMAGE_CVE","scopeQuery":"","pagination":{"offset":0,"limit":8,"sortOption":{"field":"CVE Created Time","reversed":true}}},"query":"query recentlyDetectedImageVulnerabilities($query: String, $scopeQuery: String, $pagination: Pagination) {\\n  results: imageVulnerabilities(query: $query, pagination: $pagination) {\\n    id\\n    cve\\n    cvss\\n    scoreVersion\\n    deploymentCount\\n    imageCount\\n    isFixable(query: $scopeQuery)\\n    envImpact\\n    createdAt\\n    summary\\n    __typename\\n  }\\n}\\n"}',
  'mostCommonImageVulnerabilities':            '{"operationName":"mostCommonImageVulnerabilities","variables":{"query":"","vulnPagination":{"offset":0,"limit":15,"sortOption":{"field":"Deployment Count","reversed":true}}},"query":"query mostCommonImageVulnerabilities($query: String, $vulnPagination: Pagination) {\\n  results: imageVulnerabilities(query: $query, pagination: $vulnPagination) {\\n    id\\n    cve\\n    cvss\\n    scoreVersion\\n    isFixable\\n    deploymentCount\\n    imageCount\\n    summary\\n    imageCount\\n    lastScanned\\n    __typename\\n  }\\n}\\n"}',
  'deploymentsWithMostSeverePolicyViolations': '{"operationName":"deploymentsWithMostSeverePolicyViolations","variables":{"query":"Category:Vulnerability Management","pagination":{"offset":0,"limit":8}},"query":"query deploymentsWithMostSeverePolicyViolations($query: String, $pagination: Pagination) {\\n  results: deploymentsWithMostSevereViolations(\\n    query: $query\\n    pagination: $pagination\\n  ) {\\n    id\\n    name\\n    clusterName\\n    namespaceName: namespace\\n    failingPolicySeverityCounts {\\n      critical\\n      high\\n      medium\\n      low\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
  'clustersWithMostClusterVulnerabilities':    '{"operationName":"clustersWithMostClusterVulnerabilities","variables":{},"query":"query clustersWithMostClusterVulnerabilities {\\n  results: clusters {\\n    id\\n    name\\n    isGKECluster\\n    isOpenShiftCluster\\n    clusterVulnerabilityCount\\n    clusterVulnerabilities {\\n      cve\\n      isFixable\\n      vulnerabilityType\\n      vulnerabilityTypes\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}'
}

SCOPE_DISTRIBUTION = {
  'SingleNamespace':                    {'ClusterCount':  1, 'NamespaceCount':   1},
  'TwoNamespacesInOneCluster':          {'ClusterCount':  1, 'NamespaceCount':   2},
  'TwoNamespacesInTwoClusters':         {'ClusterCount':  2, 'NamespaceCount':   2},
  'FourNamespacesInOneCluster':         {'ClusterCount':  1, 'NamespaceCount':   4},
  'FourNamespacesInTwoClusters':        {'ClusterCount':  2, 'NamespaceCount':   4},
  'FourNamespacesInFourClusters':       {'ClusterCount':  4, 'NamespaceCount':   4},
  'FiveNamespacesInOneCluster':         {'ClusterCount':  1, 'NamespaceCount':   5},
  'FiveNamespacesInTwoClusters':        {'ClusterCount':  2, 'NamespaceCount':   5},
  'FiveNamespacesInFiveClusters':       {'ClusterCount':  5, 'NamespaceCount':   5},
  'TenNamespacesInOneCluster':          {'ClusterCount':  1, 'NamespaceCount':  10},
  'TenNamespacesInTwoClusters':         {'ClusterCount':  2, 'NamespaceCount':  10},
  'TenNamespacesInFiveClusters':        {'ClusterCount':  5, 'NamespaceCount':  10},
  'TenNamespacesInTenClusters':         {'ClusterCount': 10, 'NamespaceCount':  10},
  'OneHundredNamespacesInOneCluster':   {'ClusterCount':  1, 'NamespaceCount': 100},
  'OneHundredNamespacesInFiveClusters': {'ClusterCount':  5, 'NamespaceCount': 100},
  'OneHundredNamespacesInTenClusters':  {'ClusterCount': 10, 'NamespaceCount': 100}
}

ANALYST_PERMISSION_SET = 'io.stackrox.authz.permissionset.analyst'

TEST_ROLES = {
  'Analyst':                                   {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'Unrestricted'},
  'AnalystSingleNamespace':                    {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'SingleNamespace'},
  'AnalystTwoNamespacesInOneCluster':          {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'TwoNamespacesInOneCluster'},
  'AnalystTwoNamespacesInTwoClusters':         {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'TwoNamespacesInTwoClusters'},
  'AnalystFourNamespacesInOneCluster':         {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'FourNamespacesInOneCluster'},
  'AnalystFourNamespacesInTwoClusters':        {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'FourNamespacesInTwoClusters'},
  'AnalystFourNamespacesInFourClusters':       {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'FourNamespacesInFourClusters'},
  'AnalystFiveNamespacesInOneCluster':         {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'FiveNamespacesInOneCluster'},
  'AnalystFiveNamespacesInTwoClusters':        {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'FiveNamespacesInTwoClusters'},
  'AnalystFiveNamespacesInFiveClusters':       {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'FiveNamespacesInFiveClusters'},
  'AnalystTenNamespacesInOneCluster':          {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'TenNamespacesInOneCluster'},
  'AnalystTenNamespacesInTwoClusters':         {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'TenNamespacesInTwoClusters'},
  'AnalystTenNamespacesInFiveClusters':        {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'TenNamespacesInFiveClusters'},
  'AnalystTenNamespacesInTenClusters':         {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'TenNamespacesInTenClusters'},
  'AnalystOneHundredNamespacesInOneCluster':   {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'OneHundredNamespacesInOneCluster'},
  'AnalystOneHundredNamespacesInFiveClusters': {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'OneHundredNamespacesInFiveClusters'},
  'AnalystOneHundredNamespacesInTenClusters':  {K_PERMISSIONSETID: ANALYST_PERMISSION_SET, K_ACCESSSCOPENAME: 'OneHundredNamespacesInTenClusters'}
}

RESERVED_SCOPES = {
}

##### UTILITY FUNCTIONS #####

def getPassword():
  passwd = ''
  with open(PASSWORD_FILE_PATH) as f:
    for line in f:
      passwd = line.replace('\n','').replace('\r','')
      break
  return passwd

def getRequest(endpoint, query, token, timeout = None):
  rspjson = {}
  if token == '':
    if query != None:
      rsp = requests.get(HOST+endpoint, verify=False, auth=('admin', getPassword()), data=query, timeout=timeout)
      rspjson = rsp.json()
    else:
      rsp = requests.get(HOST+endpoint, verify=False, auth=('admin', getPassword()), timeout=timeout)
      rspjson = rsp.json()
  else:
    headers = {'authorization': 'Bearer '+token}
    if query != None:
      rsp = requests.get(HOST+endpoint, verify=False, headers=headers, data=query, timeout=timeout)
      rspjson = rsp.json()
    else:
      rsp = requests.get(HOST+endpoint, verify=False, headers=headers, timeout=timeout)
      rspjson = rsp.json()
  return rspjson

def getRequestAsAdmin(endpoint, query, timeout = None):
  return getRequest(endpoint, query, '', timeout)

def postRequest(endpoint, query, token, debug=False):
  rspjson = {}
  if query != None: query=json.dumps(query)
  if token == '':
    if query != None:
      rsp = requests.post(HOST+endpoint, verify=False, auth=('admin', getPassword()), data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.post(HOST+endpoint, verify=False, auth=('admin', getPassword()))
      rspjson = rsp.json()
  else:
    headers = {'authorization': 'Bearer '+token}
    if query != None:
      rsp = requests.post(HOST+endpoint, verify=False, headers=headers, data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.post(HOST+endpoint, verify=False, headers=headers)
      rspjson = rsp.json()
  return rspjson

def putRequest(endpoint, query, token, debug=False):
  rspjson = {}
  if query != None: query=json.dumps(query)
  if token == '':
    if query != None:
      rsp = requests.put(HOST+endpoint, verify=False, auth=('admin', getPassword()), data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.put(HOST+endpoint, verify=False, auth=('admin', getPassword()))
      rspjson = rsp.json()
  else:
    headers = {'authorization': 'Bearer '+token}
    if query != None:
      rsp = requests.put(HOST+endpoint, verify=False, headers=headers, data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.put(HOST+endpoint, verify=False, headers=headers)
      rspjson = rsp.json()
  return rspjson

def deleteRequest(endpoint, token):
  if token == '':
    requests.delete(HOST+endpoint, verify=False, auth=('admin', getPassword()))
  else:
    headers = {'authorization': 'Bearer '+token}
    requests.delete(HOST+endpoint, verify=False, headers=headers)

def getElapsed(ts1, ts2):
  delta = ts2 - ts1
  if ts1 > ts2:
    delta = ts1 - ts2
  #wms = delta.weeks   * 604800000
  dms = delta.days    *  86400000
  #hms = delta.hours   *   3600000
  #mms = delta.minutes *     60000
  sms = delta.seconds *      1000
  #rms = delta.milliseconds
  rms = delta.microseconds // 1000
  #tms = rms + sms + mms + hms + dms + wms
  tms = rms + sms + dms
  grain = 5
  remain = tms % grain
  ms = tms // grain
  if remain > 0:
    ms += 1
  return ms*grain

def createNamespaceScope(namespaces, scopeName, scopeDescription):
  id = ''
  query = {
    K_NAME: scopeName,
    K_DESCRIPTION: scopeDescription,
    K_RULES: {
      K_INCLUDED_NAMESPACES: []
    }
  }
  for ns in namespaces:
    query[K_RULES][K_INCLUDED_NAMESPACES].append({K_CLUSTERNAME: ns.clusterName, K_NAMESPACENAME: ns.namespaceName})
  rspjson = postRequest(SIMPLE_ACCESS_SCOPE_ENDPOINT, query, '')
  id = rspjson[K_ID]
  return id

def createRole(rolename, permissionsetid, accessscopeid, description):
  if not PERMISSION_SET_PREFIX in permissionsetid:
    permissionsetid = PERMISSION_SET_PREFIX+permissionsetid
  if not ACCESS_SCOPE_PREFIX in accessscopeid:
    accessscopeid = ACCESS_SCOPE_PREFIX+accessscopeid
  query = {
    K_NAME: rolename,
    K_DESCRIPTION: description,
    K_RESOURCETOACCESS: {},
    K_PERMISSIONSETID: permissionsetid,
    K_ACCESSSCOPEID: accessscopeid
  }
  rspjson = postRequest(ROLE_ENDPOINT+'/'+rolename, query, '')

def getToken(role):
  token = ''
  query = {
    K_NAME: role+'_token',
    K_ROLES: [role]
  }
  rspjson = postRequest(TOKEN_GENERATOR_ENDPOINT, query, '')
  # print(json.dumps(rspjson))
  if K_TOKEN in rspjson:
    token = rspjson[K_TOKEN]
  return token

def getNamespaces():
  start = datetime.datetime.now()
  rspjson = getRequestAsAdmin(NAMESPACE_ENDPOINT, EMPTY_QUERY)
  end = datetime.datetime.now()
  elapsed = getElapsed(start, end)
  print('Namespace lookup took ' + str(end - start) + ' seconds (?)')
  print('Namespace lookup took approximately ' + str(elapsed) + ' ms')
  print(' - started : ' + str(start))
  print(' - ended   : ' + str(end))
  namespacesByClusterAndName = {}
  if not K_NAMESPACES in rspjson:
    return namespacesByClusterAndName
  for ns in rspjson[K_NAMESPACES]:
    if not K_METADATA in ns:
      continue
    if not K_NAME in ns[K_METADATA]:
      continue
    if not K_CLUSTERNAME in ns[K_METADATA]:
      continue
    clusterName = ns[K_METADATA][K_CLUSTERNAME]
    namespaceName = ns[K_METADATA][K_NAME]
    if not clusterName in namespacesByClusterAndName:
      namespacesByClusterAndName[clusterName] = {}
    namespacesByClusterAndName[clusterName][namespaceName] = ns
  return namespacesByClusterAndName

def getAccessScopes():
  rspjson = getRequestAsAdmin(SIMPLE_ACCESS_SCOPE_ENDPOINT, EMPTY_QUERY)
  accessScopesByName = {}
  ACCESS_SCOPES = 'accessScopes'
  if not ACCESS_SCOPES in rspjson:
    return accessScopesByName
  for scope in rspjson[ACCESS_SCOPES]:
    scopeName = scope[K_NAME]
    accessScopesByName[scopeName] = scope
  return accessScopesByName

def ensureScopeDistribution(scope, allAccessScopes):
  if not scope in allAccessScopes:
    print('Scope missing : ' + scope)
    return False
  distrib = SCOPE_DISTRIBUTION[scope]
  scopedata = allAccessScopes[scope]
  if not K_RULES in scopedata:
    print('No rule for scope ' + scope)
    return False
  if not 'includedNamespaces' in scopedata[K_RULES]:
    print('No namespace by name rule for scope ' + scope)
    return False
  namespaces = scopedata[K_RULES]['includedNamespaces']
  clusters = {}
  for ruledata in namespaces:
    cluster = ruledata[K_CLUSTERNAME]
    namespace = ruledata[K_NAMESPACENAME]
    if not cluster in clusters:
      clusters[cluster] = []
    clusters[cluster].append(namespace)
  if len(clusters) != distrib['ClusterCount']:
    print('Cluster count mismatch for scope [' + scope + '] (expected ' + str(distrib['ClusterCount']) + ' but got ' + str(len(clusters)) + ')')
    return False
  if len(namespaces) != distrib['NamespaceCount']:
    print('Namespace rule count mismatch for scope [' + scope + '] (expected ' + str(distrib['NamespaceCount']) + ' but got ' + str(len(namespaces)) + ')')
    return False
  return True

def ensurePredefinedAccessScope(name, accessScopes, namespacesByClusterAndName, hasPredefinedNamespaces):
  if name in accessScopes:
    scope = accessScopes[name]
    if hasPredefinedNamespaces:
      scopeNamespacesByCluster = {}
      if K_RULES in scope and 'includedNamespaces' in scope[K_RULES]:
        for rule in scope[K_RULES]['includedNamespaces']:
          ruleCluster = ''
          ruleNamespace = ''
          if K_CLUSTERNAME in rule: ruleCluster = rule[K_CLUSTERNAME]
          if K_NAMESPACENAME in rule: ruleNamespace = rule[K_NAMESPACENAME]
          if ruleCluster != '' and ruleCluster not in scopeNamespacesByCluster:
            scopeNamespacesByCluster[ruleCluster] = []
          if ruleCluster != '' and ruleNamespace != '':
            scopeNamespacesByCluster[ruleCluster].append(ruleNamespace)
      hasAllPredefinedNamespaces = True
      reservedNamespacesByCluster = {}
      for cluster in RESERVED_SCOPES:
        # print('checking namespaces from cluster ' + cluster + ' (' + str(len(RESERVED_SCOPES[cluster])) + ')')
        namespaceMatches = 0
        for namespace in RESERVED_SCOPES[cluster]:
          # if cluster in ['stackrox4', 'stackrox5']:
          #   print(cluster + ',' + namespace + '|' + RESERVED_SCOPES[cluster][namespace])
          if RESERVED_SCOPES[cluster][namespace] == name:
            namespaceMatches += 1
            if cluster not in reservedNamespacesByCluster:
              # print('scope ' + name + ' has reserved cluster ' + cluster)
              reservedNamespacesByCluster[cluster] = []
            reservedNamespacesByCluster[cluster].append(namespace)
        # print('got ' + str(namespaceMatches) + ' matching namespaces for scope ' + name + ' in cluster ' + cluster)
      if len(reservedNamespacesByCluster) != len(scopeNamespacesByCluster):
        print('cluster count mismatch between reserved and scope ' + str(len(reservedNamespacesByCluster)) + ' vs ' + str(len(scopeNamespacesByCluster)))
        hasAllPredefinedNamespaces = False
      if hasAllPredefinedNamespaces:
        for cluster in reservedNamespacesByCluster:
          if cluster not in scopeNamespacesByCluster:
            print('reserved cluster [' + cluster + '] not found in scope')
            hasAllPredefinedNamespaces = False
            break
          if len(reservedNamespacesByCluster[cluster]) != len(scopeNamespacesByCluster[cluster]):
            print('namespace count mismatch between reserved and scope ' + str(len(reservedNamespacesByCluster[cluster])) + ' vs ' + str(len(scopeNamespacesByCluster[cluster])))
            hasAllPredefinedNamespaces = False
            break
          for namespace in reservedNamespacesByCluster[cluster]:
            if namespace not in scopeNamespacesByCluster[cluster]:
              print('reserved cluster,namespace pair ('+cluster+','+namespace+'] not found in scope')
              hasAllPredefinedNamespaces = False
              break
          if not hasAllPredefinedNamespaces:
            break
      if hasAllPredefinedNamespaces:
        print('Scope ' + name + ' exists, and contains the pre-defined namespaces') 
        return
      else:
        print('Scope ' + name + ' exists, but not all predefined namespaces are in there')
        query = {K_ID: scope[K_ID], K_NAME: name, K_DESCRIPTION: '', K_RULES:{'includedNamespaces':[]}}
        for cluster in reservedNamespacesByCluster:
          for namespace in reservedNamespacesByCluster[cluster]:
            query[K_RULES]['includedNamespaces'].append({K_CLUSTERNAME: cluster, K_NAMESPACENAME: namespace})
        rsp = putRequest(SIMPLE_ACCESS_SCOPE_ENDPOINT + '/' + scope[K_ID], query, '')
        return
    else:
      print('Scope ' + name + ' exists, but not all predefined namespaces exist')
      hasCorrectClusterNamespaceDistribution = ensureScopeDistribution(scope, accessScopes)
      if hasCorrectClusterNamespaceDistribution:
        print('Scope ' + name + ' exists, has correct cluster/namespace distribution, but not all predefined namespaces exist')
      else:
        query = {K_ID: scope[K_ID], K_NAME: name, K_DESCRIPTION: '', K_RULES:{'includedNamespaces':[]}}
        # TODO
        putRequest(SIMPLE_ACCESS_SCOPE_ENDPOINT + '/' + scope[K_ID], query, '')
      return
  else:
    if hasPredefinedNamespaces:
      print('Scope ' + name + ' does not exist, but all predefined namespaces exist')
      query = {K_NAME: name, K_DESCRIPTION: '', K_RULES:{'includedNamespaces':[]}}
      for cluster in reservedNamespacesByCluster:
        for namespace in reservedNamespacesByCluster[cluster]:
          query[K_RULES]['includedNamespaces'].append({'ClusterName': cluster, K_NAMESPACENAME: namespace})
      postRequest(SIMPLE_ACCESS_SCOPE_ENDPOINT, query, '')
      return
    else:
      print('Scope ' + name + ' does not exist, but not all predefined namespaces exist')
      query = {K_ID: scope[K_ID], K_NAME: name, K_DESCRIPTION: '', K_RULES:{'includedNamespaces':[]}}
      # TODO
      postRequest(SIMPLE_ACCESS_SCOPE_ENDPOINT, query, '')
      return

def ensurePredefinedAccessScopes():
  namespacesByClusterAndName = getNamespaces()
  hasPredefinedNamespaces = True
  for cluster in RESERVED_SCOPES:
    if cluster not in namespacesByClusterAndName:
      hasPredefinedNamespaces = False
      break
    for namespace in RESERVED_SCOPES[cluster]:
      if namespace not in namespacesByClusterAndName[cluster]:
        hasPredefinedNamespaces = False
        break
    if not hasPredefinedNamespaces:
      break
  existingAccessScopes = getAccessScopes()
  for scope in SCOPE_DISTRIBUTION:
    ensurePredefinedAccessScope(scope, existingAccessScopes, namespacesByClusterAndName, hasPredefinedNamespaces)
  allAccessScopes = getAccessScopes()
  for scope in SCOPE_DISTRIBUTION:
    ensureScopeDistribution(scope, allAccessScopes)

def runTimedGraphQLQuery(opname, token):
  start = datetime.datetime.now()
  query = ''
  if opname in GRAPHQL_WIDGET_QUERIES:
    query = GRAPHQL_WIDGET_QUERIES[opname]
    try:
      result = getRequest(GRAPHQL_ENDPOINT + '?opname=' + opname, query, token, QUERY_TIMEOUT)
      # print(str(result))
    except:
      pass
  end = datetime.datetime.now()
  elapsed = getElapsed(start, end)
  # print('GraphQL query ' + opname + ' took ' + str(elapsed) + ' ms')
  return elapsed

##### MAIN SCRIPT CONTENT #####

urllib3.disable_warnings()

#ensurePredefinedAccessScopes()
allAccessScopes = getAccessScopes()
for scope in SCOPE_DISTRIBUTION:
  ensureScopeDistribution(scope, allAccessScopes)

elapsedByWidgetAndRole = {}
for it in range(TEST_RUN_COUNTS):
  runstart = datetime.datetime.now()
  for role in TEST_ROLES:
    for widget in GRAPHQL_WIDGET_QUERIES:
      # print(str(it) + ' - testing widget ' + widget + ' for role ' + role)
      sys.stdout.write('.')
      sys.stdout.flush()
      if widget not in elapsedByWidgetAndRole:
        elapsedByWidgetAndRole[widget] = {}
      if role not in elapsedByWidgetAndRole[widget]:
        elapsedByWidgetAndRole[widget][role] = []
      token = getToken(role)
      elapsed = runTimedGraphQLQuery(widget, token)
      elapsedByWidgetAndRole[widget][role].append(elapsed)
    sys.stdout.write('+')
    sys.stdout.flush()
  sys.stdout.write('\n')
  runend = datetime.datetime.now()
  runelapsed = getElapsed(runstart, runend)
  print('Run ' + str(it+1) + ' took ' + str(runelapsed) + 'ms')

widget_perf_arrays = {}
for widget in GRAPHQL_WIDGET_QUERIES:
  perf_array = []
  for role in TEST_ROLES:
    if widget not in elapsedByWidgetAndRole:
      continue
    if role not in elapsedByWidgetAndRole[widget]:
      continue
    elapsedArray = elapsedByWidgetAndRole[widget][role]
    if len(elapsedArray) <= 0:
      continue
    minelapsed = min(elapsedArray)
    totalelapsed = sum(elapsedArray)
    avgelapsed = totalelapsed // len(elapsedArray)
    maxelapsed = max(elapsedArray)
    print('Role ' + role + ' - widget ' + widget + ' elapsed min ' + str(minelapsed) + ' avg ' + str(avgelapsed) + ' max ' + str(maxelapsed))
    perf_array.append(avgelapsed)
  widget_perf_arrays[widget] = perf_array

for widget in GRAPHQL_WIDGET_QUERIES:
  perf_array = widget_perf_arrays[widget]
  print(widget + ',' + ','.join([str(x) for x in perf_array]))
