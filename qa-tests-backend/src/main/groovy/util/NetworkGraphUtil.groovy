package util

import com.google.protobuf.Timestamp
import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.NetworkFlowOuterClass
import io.stackrox.proto.storage.NetworkFlowOuterClass.NetworkEntityInfo
import objects.Edge
import io.stackrox.proto.api.v1.NetworkGraphServiceOuterClass
import services.DeploymentService
import services.NetworkGraphService

@Slf4j
class NetworkGraphUtil {

    // more time is needed on few architectures
    static final NETWORK_FLOW_UPDATE_CADENCE_IN_SECONDS =
        ((Env.REMOTE_CLUSTER_ARCH == "x86_64" ) ? 30 : 120)

    static int edgeCount(NetworkGraphServiceOuterClass.NetworkGraph graph) {
        int numEdges = 0
        graph.nodesList.each {
            numEdges += it.outEdgesCount
        }
        return numEdges
    }

    static Set<String> deployments(NetworkGraphServiceOuterClass.NetworkGraph graph) {
        def deploymentSet = new HashSet<String>([])

        graph.nodesList.each {
            if (it.entity.type != NetworkFlowOuterClass.NetworkEntityInfo.Type.DEPLOYMENT) {
                return
            }
            deploymentSet.add("${it.entity.deployment.namespace}/${it.entity.deployment.name}")
        }
        return deploymentSet
    }

    private static String entityLabel(NetworkEntityInfo entity) {
        if (entity.type == NetworkFlowOuterClass.NetworkEntityInfo.Type.DEPLOYMENT) {
            return "${entity.deployment.namespace}/${entity.deployment.name}"
        } else if (entity.type == NetworkFlowOuterClass.NetworkEntityInfo.Type.INTERNET) {
            return "INTERNET"
        }
        return ""
    }

    static Set<String> flowStrings(NetworkGraphServiceOuterClass.NetworkGraph graph) {
        return new HashSet<String>(graph.nodesList.<String>collectMany {
            def srcLabel = entityLabel(it.entity)
            return srcLabel ? it.outEdges.collectMany {
                def tgt = graph.nodesList.get(it.key)
                def dstLabel = entityLabel(tgt.entity)
                return dstLabel ? ["${srcLabel} -> ${dstLabel}"] : []
            } : []
        })
    }

    static NetworkGraphServiceOuterClass.NetworkNode findDeploymentNode(
            NetworkGraphServiceOuterClass.NetworkGraph graph, String deploymentId) {
        return graph.nodesList.find {
            it.deploymentId == deploymentId
        }
    }

    static List<Edge> findEdges(NetworkGraphServiceOuterClass.NetworkGraph graph, String sourceId, String targetId) {
        log.debug "Checking for edge between deployments: sourceId ${sourceId}, targetId ${targetId}"

        def sourceNodes = sourceId == null ? graph.nodesList : graph.nodesList.findAll {
            it.deploymentId == sourceId
        }
        def targetNodeIndex = graph.nodesList.findIndexOf {
            it.deploymentId == targetId
        }

        if ((sourceId != null && sourceNodes.empty) || (targetId != null && targetNodeIndex == -1)) {
            if (sourceId != null && sourceNodes.empty) {
                log.debug "Found no nodes matching sourceId ${sourceId}"
            }
            if (targetId != null && targetNodeIndex == -1) {
                log.debug "Found no nodes matching targetId ${targetId}"
            }
            return []
        }

        log.debug "Looking at edges for ${sourceNodes.size()} source node(s)"

        return sourceNodes.collectMany {
            def currentSourceId = it.deploymentId
            return it.getOutEdgesMap().collectMany {
                if (targetNodeIndex != -1 && it.key != targetNodeIndex) {
                    return []
                }
                log.debug "Source Id ${currentSourceId} -> edge target key: ${it.key}"
                def targetNode = graph.nodesList.get(it.key)
                log.debug "  -> targetId: ${targetNode.deploymentId}"

                def props = it.value.propertiesList
                props.forEach {
                    edgeProp -> log.debug "    -> edge: ${edgeProp.port} ${edgeProp.protocol} "+
                            "${edgeProp.lastActiveTimestamp.seconds}.${edgeProp.lastActiveTimestamp.nanos}"
                }
                if (props == null || props.empty) {
                    props = [null]
                }
                props.collect {
                    new Edge(sourceID: currentSourceId, targetID: targetNode.deploymentId, edgeProperties: it)
                }
            }
        }
    }

    static checkForEdge(String sourceId, String targetId, Timestamp since = null,
                        int timeoutSeconds = 120, String query = null) {
        int intervalSeconds = 1
        int waitTime
        def startTime = System.currentTimeMillis()
        for (waitTime = 0; waitTime <= timeoutSeconds / intervalSeconds; waitTime++) {
            if (waitTime > 0) {
                sleep intervalSeconds * 1000
            }

            def graph = NetworkGraphService.getNetworkGraph(since, query)
            def edges = NetworkGraphUtil.findEdges(graph, sourceId, targetId)
            if (edges != null && edges.size() > 0) {
                log.debug "Found source ${sourceId} -> target ${targetId} " +
                    "in graph after ${(System.currentTimeMillis() - startTime) / 1000}s"
                return edges
            }
        }
        log.warn "SR did not detect the edge in Network Flow graph"
        return null
    }

    static NetworkGraphNodes getDeploymentsAsGraphNodes() {
        def deployments = DeploymentService.listDeploymentsSearch(
                SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Orchestrator Component:true").build()
        ).deploymentsList
        Set<String> orchestratorDeployments = new HashSet<String>([])
        deployments.each { orchestratorDeployments.add("${it.namespace}/${it.name}") }

        deployments = DeploymentService.listDeploymentsSearch(
                SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Orchestrator Component:false").build()
        ).deploymentsList
        Set<String> nonOrchestratorDeployments = new HashSet<String>([])
        deployments.each { nonOrchestratorDeployments.add("${it.namespace}/${it.name}") }

        return new NetworkGraphNodes(orchestratorDeployments, nonOrchestratorDeployments)
    }

    static boolean verifyGraphFilterAndScope(
            NetworkGraphServiceOuterClass.NetworkGraph graph,
            Set<String> nonOrchestratorDeployments,
            Set<String> orchestratorDeployments,
            boolean nonOrchestratorComponentsShouldExist,
            boolean orchestratorComponentsShouldExist
    ) {
        def graphDeployments = deployments(graph)
        assert nonOrchestratorComponentsShouldExist ==
                (nonOrchestratorDeployments.intersect(graphDeployments).size() > 0)
        assert orchestratorComponentsShouldExist ==
                (orchestratorDeployments.intersect(graphDeployments).size() > 0)
        return true
    }

    static class NetworkGraphNodes {
        Set<String> orchestratorDeployments
        Set<String> nonOrchestratorDeployments

        NetworkGraphNodes(Set<String> orchestratorDeployments, Set<String> nonOrchestratorDeployments) {
            this.orchestratorDeployments = orchestratorDeployments
            this.nonOrchestratorDeployments = nonOrchestratorDeployments
        }
    }
}

