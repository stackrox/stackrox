package objects

import com.google.protobuf.util.Timestamps
import io.stackrox.proto.api.v1.NetworkGraphOuterClass.NetworkEdgeProperties

class Edge {
    String sourceID
    String targetID
    NetworkEdgeProperties edgeProperties

    def getLastActiveTimestamp() { Timestamps.toMillis(edgeProperties?.lastActiveTimestamp) }
    def getProtocol() { edgeProperties?.protocol }
    def getPort() { edgeProperties?.port }
}
