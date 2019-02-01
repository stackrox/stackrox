package objects

import io.stackrox.proto.storage.Compliance

class Control {
    def id
    def evidenceMessages = []
    def state
    def standard
    def type = ControlType.DEPLOYMENT

    Control(String i, List<String> em, Compliance.ComplianceState s) {
        id = i
        evidenceMessages = em
        state = s
        standard = i.split(":")[0]
    }

    def setType(ControlType t) {
        type = t
        return this
    }

    enum ControlType {
        CLUSTER,
        NODE,
        DEPLOYMENT
    }
}
