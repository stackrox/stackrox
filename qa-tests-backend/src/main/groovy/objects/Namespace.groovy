package objects

import groovy.transform.CompileStatic

@CompileStatic
class Namespace {
    String uid
    String name
    Map<String, String> labels
    List<String> deploymentCount
    int secretsCount
    int networkPolicyCount
}
