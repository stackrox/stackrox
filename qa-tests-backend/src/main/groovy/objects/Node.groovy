package objects

import groovy.transform.CompileStatic

@CompileStatic
class Node {
    String uid
    String name
    Map<String, String> labels
    Map<String, String> annotations
    List<String> internalIps
    List<String> externalIps
    String containerRuntimeVersion
    String kernelVersion
    String osImage
    String kubeletVersion
    String kubeProxyVersion
}
