package util

import objects.Edge
import v1.NetworkGraphOuterClass

class NetworkGraphUtil {

    static List<Edge> findEdges(NetworkGraphOuterClass.NetworkGraph graph, String sourceId, String targetId) {
        def sourceNodes = sourceId == null ? graph.nodesList : graph.nodesList.findAll {
            it.deploymentId == sourceId
        }
        def targetNodeIndex = graph.nodesList.findIndexOf {
            it.deploymentId == targetId
        }

        if ((sourceId != null && sourceNodes.empty) || (targetId != null && targetNodeIndex == -1)) {
            return []
        }
        return sourceNodes.collectMany {
            def currentSourceId = it.deploymentId
            return it.getOutEdgesMap().collectMany {
                if (targetNodeIndex != -1 && it.key != targetNodeIndex) {
                    return []
                }
                def targetNode = graph.nodesList.get(it.key)

                def props = it.value.propertiesList
                if (props == null || props.empty) {
                    props = [null]
                }
                props.collect {
                    new Edge(sourceID: currentSourceId, targetID: targetNode.deploymentId, edgeProperties: it)
                }
            }
        }
    }

}

