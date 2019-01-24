package objects

import io.stackrox.proto.storage.Compliance

class Control {
    def id
    def evidenceMessages = []
    def success

    Control(String i, List<String> em, Compliance.ComplianceState s) {
        id = i
        evidenceMessages = em
        success = s
    }
}
