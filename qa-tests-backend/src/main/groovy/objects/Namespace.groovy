package objects

import groovy.transform.CompileStatic

@CompileStatic
class Namespace {
    String uid
    String name
    Map<String, String> labels
    List<String> deployments
    int secretsCount
    int networkPolicyCount
}
