package objects

import groovy.transform.CompileStatic

@CompileStatic
class ConfigMap {
    String name
    String namespace
    Map<String, String> data
}
