import argparse
import datetime
import json
import requests
import sys
import time
import urllib3

##### CONSTANTS #####

GRAPHQL_ENDPOINT             = '/api/graphql'
NAMESPACE_ENDPOINT           = '/v1/namespaces'
PERMISSION_SET_ENDPOINT      = '/v1/permissionsets'
ROLE_ENDPOINT                = '/v1/roles'
SIMPLE_ACCESS_SCOPE_ENDPOINT = '/v1/simpleaccessscopes'
TOKEN_GENERATOR_ENDPOINT     = '/v1/apitokens/generate'

ACCESS_SCOPE_PREFIX = 'io.stackrox.authz.accessscope.'
PERMISSION_SET_PREFIX = 'io.stackrox.authz.permissionset.'

EMPTY_QUERY = ''

# MAP KEYS
K_ACCESSSCOPEID       = 'accessScopeId'
K_ACCESSSCOPENAME     = 'accessScopeName'
K_ACCESSSCOPES        = 'accessScopes'
K_CLUSTER_NAME        = 'cluster_name'
K_CLUSTERNAME         = 'clusterName'
K_DATA                = 'data'
K_DESCRIPTION         = 'description'
K_ID                  = 'id'
K_INCLUDED_NAMESPACES = 'included_namespaces'
K_INCLUDEDNAMESPACES  = 'includedNamespaces'
K_METADATA            = 'metadata'
K_NAME                = 'name'
K_NAMESPACENAME       = 'namespaceName'
K_NAMESPACES          = 'namespaces'
K_OPERATIONNAME       = 'operationName'
K_PERMISSIONSETS      = 'permissionSets'
K_PERMISSIONSETID     = 'permissionSetId'
K_RESOURCETOACCESS    = 'resourceToAccess'
K_ROLES               = 'roles'
K_RULES               = 'rules'
K_TOKEN               = 'token'

QUERY_TIMEOUT = 1200

GRAPHQL_WIDGET_QUERIES = {
  'summary_counts':            '{"operationName":"summary_counts","variables":{},"query":"query summary_counts {\\n  clusterCount\\n  nodeCount\\n  violationCount\\n  deploymentCount\\n  imageCount\\n  secretCount\\n}\\n"}',
  'getAllNamespacesByCluster': '{"operationName":"getAllNamespacesByCluster","variables":{"query":""},"query":"query getAllNamespacesByCluster($query: String) {\\n  clusters(query: $query) {\\n    id\\n    name\\n    namespaces {\\n      metadata {\\n        id\\n        name\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
  'mostRecentAlerts':          '{"operationName":"mostRecentAlerts","variables":{"query":"Severity:CRITICAL_SEVERITY"},"query":"query mostRecentAlerts($query: String) {\\n  alerts: violations(\\n    query: $query\\n    pagination: {limit: 3, sortOption: {field: \\"Violation Time\\", reversed: true}}\\n  ) {\\n    id\\n    time\\n    deployment {\\n      clusterName\\n      namespace\\n      name\\n      __typename\\n    }\\n    policy {\\n      name\\n      severity\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
  'getImagesDashboard':        '{"operationName":"getImages","variables":{"query":""},"query":"query getImages($query: String) {\\n  images(\\n    query: $query\\n    pagination: {limit: 6, sortOption: {field: \\"Image Risk Priority\\", reversed: false}}\\n  ) {\\n    id\\n    name {\\n      remote\\n      fullName\\n      __typename\\n    }\\n    priority\\n    imageVulnerabilityCounter {\\n      important {\\n        total\\n        fixable\\n        __typename\\n      }\\n      critical {\\n        total\\n        fixable\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
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

##### UTILITY FUNCTIONS #####

def getPassword(reqOptions):
  passwd = ''
  with open(reqOptions.passwordfile) as f:
    for line in f:
      passwd = line.replace('\n','').replace('\r','')
      break
  return passwd

def getHost(reqOptions):
  return 'https://'+reqOptions.host+':'+str(reqOptions.port)

def getRequest(endpoint, query, token, reqOptions, timeout = None):
  rspjson = {}
  host = getHost(reqOptions)
  if token == '':
    passwd = getPassword(reqOptions)
    if query != None:
      rsp = requests.get(host+endpoint, verify=False, auth=('admin', passwd), data=query, timeout=timeout)
      rspjson = rsp.json()
    else:
      rsp = requests.get(host+endpoint, verify=False, auth=('admin', passwd), timeout=timeout)
      rspjson = rsp.json()
  else:
    headers = {'authorization': 'Bearer '+token}
    if query != None:
      rsp = requests.get(host+endpoint, verify=False, headers=headers, data=query, timeout=timeout)
      rspjson = rsp.json()
    else:
      rsp = requests.get(host+endpoint, verify=False, headers=headers, timeout=timeout)
      rspjson = rsp.json()
  return rspjson

def getRequestAsAdmin(endpoint, query, reqOptions, timeout = None):
  return getRequest(endpoint, query, '', reqOptions, timeout = timeout)

def postRequest(endpoint, query, token, reqOptions, debug=False):
  rspjson = {}
  host = getHost(reqOptions)
  if query != None: query=json.dumps(query)
  if token == '':
    passwd = getPassword(reqOptions)
    if query != None:
      rsp = requests.post(host+endpoint, verify=False, auth=('admin', passwd), data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.post(host+endpoint, verify=False, auth=('admin', passwd))
      rspjson = rsp.json()
  else:
    headers = {'authorization': 'Bearer '+token}
    if query != None:
      rsp = requests.post(host+endpoint, verify=False, headers=headers, data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.post(host+endpoint, verify=False, headers=headers)
      rspjson = rsp.json()
  return rspjson

def putRequest(endpoint, query, token, reqOptions, debug=False):
  rspjson = {}
  host = getHost(reqOptions)
  if query != None: query=json.dumps(query)
  if token == '':
    passwd = getPassword(reqOptions)
    if query != None:
      rsp = requests.put(host+endpoint, verify=False, auth=('admin', passwd), data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.put(host+endpoint, verify=False, auth=('admin', passwd))
      rspjson = rsp.json()
  else:
    headers = {'authorization': 'Bearer '+token}
    if query != None:
      rsp = requests.put(host+endpoint, verify=False, headers=headers, data=query)
      rspjson = rsp.json()
    else:
      rsp = requests.put(host+endpoint, verify=False, headers=headers)
      rspjson = rsp.json()
  return rspjson

def deleteRequest(endpoint, token, reqOptions):
  host = getHost(reqOptions)
  passwd = getPassword(reqOptions)
  if token == '':
    requests.delete(host+endpoint, verify=False, auth=('admin', passwd))
  else:
    headers = {'authorization': 'Bearer '+token}
    requests.delete(host+endpoint, verify=False, headers=headers)

def getElapsed(ts1, ts2):
  delta = ts2 - ts1
  if ts1 > ts2:
    delta = ts1 - ts2
  dms = delta.days    *  86400000
  sms = delta.seconds *      1000
  rms = delta.microseconds // 1000
  tms = rms + sms + dms
  grain = 5
  remain = tms % grain
  ms = tms // grain
  if remain > 0:
    ms += 1
  return ms*grain

def createNamespaceScope(namespaces, scopeName, scopeDescription, reqOptions):
  id = ''
  query = {
    K_NAME: scopeName,
    K_DESCRIPTION: scopeDescription,
    K_RULES: {
      K_INCLUDED_NAMESPACES: []
    }
  }
  for ns in namespaces:
    query[K_RULES][K_INCLUDED_NAMESPACES].append({K_CLUSTERNAME: ns[K_METADATA][K_CLUSTERNAME], K_NAMESPACENAME: ns[K_METADATA][K_NAME]})
  rspjson = postRequest(SIMPLE_ACCESS_SCOPE_ENDPOINT, query, '', reqOptions)
  id = rspjson[K_ID]
  return id

def updateNamespaceScope(namespaces, scopeId, scopeName, scopeDescription, reqOptions):
  query = {
    K_ID: scopeId,
    K_NAME: scopeName,
    K_DESCRIPTION: scopeDescription,
    K_RULES: {
      K_INCLUDED_NAMESPACES: []
    }
  }
  for ns in namespaces:
    query[K_RULES][K_INCLUDED_NAMESPACES].append({K_CLUSTERNAME: ns[K_METADATA][K_CLUSTERNAME], K_NAMESPACENAME: ns[K_METADATA][K_NAME]})
  rspjson = putRequest(SIMPLE_ACCESS_SCOPE_ENDPOINT+'/'+scopeId, query, '', reqOptions)
  return scopeId

def createRole(rolename, permissionsetid, accessscopeid, description, reqOptions):
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
  rspjson = postRequest(ROLE_ENDPOINT+'/'+rolename, query, '', reqOptions)
  return rspjson

def updateRole(rolename, permissionsetid, accessscopeid, description, reqOptions):
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
  rspjson = putRequest(ROLE_ENDPOINT+'/'+rolename, query, '', reqOptions)
  return rspjson

def getToken(role, reqOptions):
  token = ''
  query = {
    K_NAME: role+'_token',
    K_ROLES: [role]
  }
  rspjson = postRequest(TOKEN_GENERATOR_ENDPOINT, query, '', reqOptions)
  # print(json.dumps(rspjson))
  if K_TOKEN in rspjson:
    token = rspjson[K_TOKEN]
  return token

def getNamespaces(reqOptions):
  print('Fetching namespaces')
  start = datetime.datetime.now()
  rspjson = getRequestAsAdmin(NAMESPACE_ENDPOINT, EMPTY_QUERY, reqOptions)
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

def getRoles(reqOptions):
  rspjson = getRequestAsAdmin(ROLE_ENDPOINT, EMPTY_QUERY, reqOptions)
  rolesByName = {}
  if K_ROLES not in rspjson:
    return rolesByName
  for role in rspjson[K_ROLES]:
    roleName = role[K_NAME]
    rolesByName[roleName] = role
  return rolesByName


def getAccessScopes(reqOptions):
  rspjson = getRequestAsAdmin(SIMPLE_ACCESS_SCOPE_ENDPOINT, EMPTY_QUERY, reqOptions)
  accessScopesByName = {}
  if not K_ACCESSSCOPES in rspjson:
    return accessScopesByName
  for scope in rspjson[K_ACCESSSCOPES]:
    scopeName = scope[K_NAME]
    accessScopesByName[scopeName] = scope
  return accessScopesByName

def getPermissionSets(reqOptions):
  rspjson = getRequestAsAdmin(PERMISSION_SET_ENDPOINT, EMPTY_QUERY, reqOptions)
  permissionSetsById= {}
  if not K_PERMISSIONSETS in rspjson:
    return permissionSetsById
  for permissionSet in rspjson[K_PERMISSIONSETS]:
    permissionSetId = permissionSet[K_ID]
    permissionSetsById[permissionSetId] = permissionSet
  return permissionSetsById

def checkScopeDistribution(scope, allAccessScopes):
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

def listNamespacesToReserve(clusterCount, namespaceCount, reservedNamespacesByCluster, allNamespacesByCluster):
  print('Listing ' + str(namespaceCount) + ' namespaces in ' + str(clusterCount) + ' clusters')
  if clusterCount == 0 or namespaceCount == 0:
    return {}
  namespacesPerCluster = namespaceCount // clusterCount
  remainingNamespaces = namespaceCount % clusterCount
  reservedNamespaceCountByCluster = {}
  for cluster in allNamespacesByCluster:
    reservedCount = 0
    if cluster in reservedNamespacesByCluster:
      reservedCount = len(reservedNamespacesByCluster[cluster])
    reservedNamespaceCountByCluster[cluster] = reservedCount
  clustersToUse = []
  for cluster in allNamespacesByCluster:
    if len(clustersToUse) == 0:
      clustersToUse.append(cluster)
    else:
      reservedNamespaces = reservedNamespaceCountByCluster[cluster]
      insertIndex = 0
      for ix in range(len(clustersToUse)):
        cl = clustersToUse[ix]
        reserved = 0
        if cl in reservedNamespacesByCluster:
          reserved = len(reservedNamespacesByCluster[cl])
        if reserved >= reservedNamespaces:
          insertIndex = ix
          break
      prefixClusters = clustersToUse[0:insertIndex]
      suffixClusters = clustersToUse[insertIndex:]
      ncl = []
      for cl in prefixClusters: ncl.append(cl)
      ncl.append(cluster)
      for cl in suffixClusters: ncl.append(cl)
      clustersToUse = ncl
  clustersToUse = clustersToUse[0:clusterCount]
  newlyReserved = []
  for cluster in clustersToUse:
    if cluster not in allNamespacesByCluster:
      print('Selection error: cluster ' + cluster + ' not found in all namespace list')
      continue
    namespacesToInclude = namespacesPerCluster
    if remainingNamespaces:
      namespacesToInclude += 1
      remainingNamespaces -= 1
    for nsname in allNamespacesByCluster[cluster]:
      ns = allNamespacesByCluster[cluster][nsname]
      if cluster in reservedNamespacesByCluster and nsname in reservedNamespacesByCluster[cluster]: continue
      newlyReserved.append(ns)
      namespacesToInclude -= 1
      if namespacesToInclude == 0: break
  print('Shortlisted ' + str(len(newlyReserved)) + ' namespaces')
  return newlyReserved

def ensureAccessScopes(reqOptions):
  allAccessScopes = getAccessScopes(reqOptions)
  scopesToCreate = []
  scopesToUpdate = []
  reservedNamespacesByCluster = {}
  for scope in SCOPE_DISTRIBUTION:
    if not scope in allAccessScopes:
      scopesToCreate.append(scope)
    else:
      hasProperDistribution = checkScopeDistribution(scope, allAccessScopes)
      if hasProperDistribution:
        actualScope = allAccessScopes[scope]
        for namespaceRule in actualScope[K_RULES][K_INCLUDEDNAMESPACES]:
          ruleCluster = namespaceRule[K_CLUSTERNAME]
          ruleNamespace = namespaceRule[K_NAMESPACENAME]
          if ruleCluster not in reservedNamespacesByCluster:
            reservedNamespacesByCluster[ruleCluster] = {}
          reservedNamespacesByCluster[ruleCluster][ruleNamespace] = True
      else:
        scopesToUpdate.append(scope)
  if len(scopesToCreate) == 0 and len(scopesToUpdate) == 0:
    return
  print(str(len(scopesToCreate)) + ' scopes to create and ' + str(len(scopesToUpdate)) + ' scopes to update.')
  startNamespaceLookup = datetime.datetime.now()
  allNamespaces = getNamespaces(reqOptions)
  endNamespaceLookup = datetime.datetime.now()
  elapsedNamespaceLookup = getElapsed(startNamespaceLookup, endNamespaceLookup)
  for scope in scopesToCreate:
    distribution = SCOPE_DISTRIBUTION[scope]
    scopeNamespaces = listNamespacesToReserve(distribution['ClusterCount'], distribution['NamespaceCount'], reservedNamespacesByCluster, allNamespaces)
    createNamespaceScope(scopeNamespaces, scope, '', reqOptions)
    for ns in scopeNamespaces:
      clusterName = ns[K_METADATA][K_CLUSTERNAME]
      namespaceName = ns[K_METADATA][K_NAME]
      if clusterName not in reservedNamespacesByCluster:
        reservedNamespacesByCluster[clusterName] = {}
      reservedNamespacesByCluster[clusterName][namespaceName] = True
  for scope in scopesToUpdate:
    distribution = SCOPE_DISTRIBUTION[scope]
    actualScope = allAccessScopes[scope]
    scopeId = actualScope[K_ID]
    scopeNamespaces = listNamespacesToReserve(distribution['ClusterCount'], distribution['NamespaceCount'], reservedNamespacesByCluster, allNamespaces)
    updateNamespaceScope(scopeNamespaces, scopeId, scope, '', reqOptions)
    for ns in scopeNamespaces:
      clusterName = ns[K_METADATA][K_CLUSTERNAME]
      namespaceName = ns[K_METADATA][K_NAME]
      if clusterName not in reservedNamespacesByCluster:
        reservedNamespacesByCluster[clusterName] = {}
      reservedNamespacesByCluster[clusterName][namespaceName] = True

def ensureTestRoles(reqOptions):
  ensureAccessScopes(reqOptions)
  allAccessScopes = getAccessScopes(reqOptions)
  allPermissionSets = getPermissionSets(reqOptions)
  allRoles = getRoles(reqOptions)
  rolesToCreate = []
  rolesToUpdate = []
  for role in TEST_ROLES:
    if role not in allRoles:
      rolesToCreate.append(role)
    else:
      fetchedRole = allRoles[role]
      fetchedPermissionSetId = fetchedRole[K_PERMISSIONSETID]
      rolePermissionSetId = TEST_ROLES[role][K_PERMISSIONSETID]
      if fetchedPermissionSetId != rolePermissionSetId:
        print('updating role ' + role + ' - permissionset mismatch ' + fetchedPermissionSetId + ' vs ' + rolePermissionSetId)
        rolesToUpdate.append(role)
      fetchedAccessScopeId = fetchedRole[K_ACCESSSCOPEID]
      roleAccessScopeName = TEST_ROLES[role][K_ACCESSSCOPENAME]
      roleAccessScopeId = allAccessScopes[roleAccessScopeName][K_ID]
      if fetchedRole[K_ACCESSSCOPEID] != roleAccessScopeId:
        print('updating role ' + role + ' - accessScopeId mismatch ' + fetchedAccessScopeId + ' vs ' + roleAccessScopeId)
        rolesToUpdate.append(role)
  print(str(len(rolesToCreate)) + ' roles to create and ' + str(len(rolesToUpdate)) + ' roles to update')
  for role in rolesToCreate:
    rolePermissionSetId = TEST_ROLES[role][K_PERMISSIONSETID]
    roleAccessScopeName = TEST_ROLES[role][K_ACCESSSCOPENAME]
    roleAccessScopeId = allAccessScopes[roleAccessScopeName][K_ID]
    createRole(role, rolePermissionSetId, roleAccessScopeId, '', reqOptions)
  for role in rolesToUpdate:
    rolePermissionSetId = TEST_ROLES[role][K_PERMISSIONSETID]
    roleAccessScopeName = TEST_ROLES[role][K_ACCESSSCOPENAME]
    roleAccessScopeId = allAccessScopes[roleAccessScopeName][K_ID]
    updateRole(role, rolePermissionSetId, roleAccessScopeId, '', reqOptions)

def runTimedGraphQLQuery(opname, token, reqOptions):
  start = datetime.datetime.now()
  query = ''
  result = {}
  if opname in GRAPHQL_WIDGET_QUERIES:
    query = GRAPHQL_WIDGET_QUERIES[opname]
    realOpName = opname
    parsedQuery = json.loads(query)
    if K_OPERATIONNAME in parsedQuery:
      realOpName = parsedQuery[K_OPERATIONNAME]
    try:
      result = getRequest(GRAPHQL_ENDPOINT + '?opname=' + realOpName, query, token, reqOptions, timeout = QUERY_TIMEOUT)
      # print(str(result))
    except:
      print('runTimedGraphQLQuery failure for target ' + opname + ' (' + realOpName + ')')
      pass
  end = datetime.datetime.now()
  elapsed = getElapsed(start, end)
  # print('GraphQL query ' + opname + ' took ' + str(elapsed) + ' ms')
  return elapsed, result

def collectPerformanceData(runCount, reqOptions):
  dataByWidgetAndRole = {}
  role_api_tokens = {}
  for role in TEST_ROLES:
    role_api_tokens[role] = getToken(role, reqOptions)
  for role in TEST_ROLES:
    token = role_api_tokens[role]
    sys.stdout.write('-')
    sys.stdout.flush()
    # Get Alert count
    widgetName = 'alert_count'
    elapsed, graphQLResult = runTimedGraphQLQuery('summary_counts', token, reqOptions)
    sys.stdout.write('.')
    sys.stdout.flush()
    if widgetName not in dataByWidgetAndRole:
      dataByWidgetAndRole[widgetName] = {}
    valueToAdd = 0
    if K_DATA in graphQLResult and 'violationCount' in graphQLResult[K_DATA]:
      valueToAdd = graphQLResult[K_DATA]['violationCount']
    else:
      print('No value for widget [' + widgetName + '] and role ' + role)
      print(graphQLResult)
    dataByWidgetAndRole[widgetName][role] = valueToAdd
    # Get CVE count
    widgetName = 'vulnerability_count'
    elapsed, graphQLResult = runTimedGraphQLQuery('cvesCount', token, reqOptions)
    sys.stdout.write('.')
    sys.stdout.flush()
    if widgetName not in dataByWidgetAndRole:
      dataByWidgetAndRole[widgetName] = {}
    valueToAdd = 0
    if K_DATA in graphQLResult and 'vulnerabilityCount' in graphQLResult[K_DATA]:
      valueToAdd = graphQLResult[K_DATA]['vulnerabilityCount']
    elif K_DATA in graphQLResult and ('imageVulnerabilityCount' in graphQLResult[K_DATA] or 'nodeVulnerabilityCount' in graphQLResult[K_DATA] or 'clusterVulnerabilityCount' in graphQLResult[K_DATA]):
      data = graphQLResult[K_DATA]
      for key in ['imageVulnerabilityCount', 'nodeVulnerabilityCount', 'clusterVulnerabilityCount']:
        if key in data:
          valueToAdd += data[key]
    else:
      print('No value for widget [' + widgetName + '] and role ' + role)
      print(graphQLResult)
    dataByWidgetAndRole[widgetName][role] = valueToAdd
    # Get Image count
    widgetName = 'image_count'
    elapsed, graphQLResult = runTimedGraphQLQuery('getImages', token, reqOptions)
    sys.stdout.write('.')
    sys.stdout.flush()
    if widgetName not in dataByWidgetAndRole:
      dataByWidgetAndRole[widgetName] = {}
    valueToAdd = 0
    if K_DATA in graphQLResult and 'imageCount' in graphQLResult[K_DATA]:
      valueToAdd = graphQLResult[K_DATA]['imageCount']
    else:
      print('No value for widget [' + widgetName + '] and role ' + role)
      print(graphQLResult)
    dataByWidgetAndRole[widgetName][role] = valueToAdd
    sys.stdout.write('+')
    sys.stdout.flush()
  sys.stdout.write('\n')
  sys.stdout.flush()
  for it in range(runCount):
    runstart = datetime.datetime.now()
    for role in TEST_ROLES:
      for widget in GRAPHQL_WIDGET_QUERIES:
        # print(str(it) + ' - testing widget ' + widget + ' for role ' + role)
        sys.stdout.write('.')
        sys.stdout.flush()
        if widget not in dataByWidgetAndRole:
          dataByWidgetAndRole[widget] = {}
        if role not in dataByWidgetAndRole[widget]:
          dataByWidgetAndRole[widget][role] = []
        token = role_api_tokens[role]
        elapsed, graphQLResult = runTimedGraphQLQuery(widget, token, reqOptions)
        dataByWidgetAndRole[widget][role].append(elapsed)
      sys.stdout.write('+')
      sys.stdout.flush()
    sys.stdout.write('\n')
    runend = datetime.datetime.now()
    runelapsed = getElapsed(runstart, runend)
    print('Run ' + str(it+1) + ' took ' + str(runelapsed) + 'ms')
  return dataByWidgetAndRole

def displayPerformanceData(dataByWidgetAndRole):
  widget_perf_arrays = {}
  for widget in ['alert_count', 'vulnerability_count', 'image_count']:
    perf_array = []
    for role in TEST_ROLES:
      valueToAppend = 0
      if widget in dataByWidgetAndRole and role in dataByWidgetAndRole[widget]:
        valueToAppend = dataByWidgetAndRole[widget][role]
      else:
        print('No value for widget ' + widget + ' and role ' + role)
      perf_array.append(valueToAppend)
    widget_perf_arrays[widget] = perf_array
  for widget in GRAPHQL_WIDGET_QUERIES:
    perf_array = []
    for role in TEST_ROLES:
      if widget not in dataByWidgetAndRole:
        continue
      if role not in dataByWidgetAndRole[widget]:
        continue
      elapsedArray = dataByWidgetAndRole[widget][role]
      if len(elapsedArray) <= 0:
        continue
      minelapsed = min(elapsedArray)
      totalelapsed = sum(elapsedArray)
      avgelapsed = totalelapsed // len(elapsedArray)
      maxelapsed = max(elapsedArray)
      print('Role ' + role + ' - widget ' + widget + ' elapsed min ' + str(minelapsed) + ' avg ' + str(avgelapsed) + ' max ' + str(maxelapsed))
      # TODO: consider taking the average augmented or diminished by the standard deviation
      perf_array.append(avgelapsed)
    widget_perf_arrays[widget] = perf_array

  for widget in ['alert_count', 'vulnerability_count', 'image_count']:
    perf_array = widget_perf_arrays[widget]
    print(widget + ',' + ','.join([str(x) for x in perf_array]))
  for widget in GRAPHQL_WIDGET_QUERIES:
    perf_array = widget_perf_arrays[widget]
    print(widget + ',' + ','.join([str(x) for x in perf_array]))

def parseOptions():
  parser = argparse.ArgumentParser()
  parser.add_argument("-H", "--host", help="host to send the queries to.", type=str, default="localhost")
  parser.add_argument("-P", "--port", help="port to send the queries to on the target host.", type=int, default=8000)
  parser.add_argument("-C", "--count", help="number of runs for statistical value extraction", type=int, default=10)
  parser.add_argument("-F", "--passwordfile", help="path to the password file", type=str, default="deploy/k8s/central-deploy/password")
  opts = parser.parse_args()
  return opts

##### MAIN SCRIPT CONTENT #####

options = parseOptions()

urllib3.disable_warnings()

ensureTestRoles(options)

dataByWidgetAndRole = collectPerformanceData(options.count, options)

displayPerformanceData(dataByWidgetAndRole)
