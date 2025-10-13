package objects

import groovy.transform.CompileStatic

@CompileStatic
class Secret {
    String name
    String namespace
    String username
    String password
    String server
    Map<String, String> data
    String type
}
