package services

import groovy.util.logging.Slf4j

import io.stackrox.proto.api.v1.CollectionServiceGrpc
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.ResourceCollectionService
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ResourceCollectionOuterClass

@Slf4j
class CollectionsService extends BaseService {

    static getClient() {
        return CollectionServiceGrpc.newBlockingStub(getChannel())
    }

    // Create a collection that matches the chosen deployment in the chosen namespaces
    // If no deployments or ns are provided, select all.
    // TODO: support embedded, regex, labels
    static ResourceCollectionOuterClass.ResourceCollection createCollection(List<String> deployments,
                                                                            List<String> namespaces) {
        def selector = ResourceCollectionOuterClass.ResourceSelector.newBuilder()

        if (deployments.size() > 0) {
            def rule = ResourceCollectionOuterClass.SelectorRule.newBuilder()
                    .setFieldName("Deployment")
                    .setOperator(PolicyOuterClass.BooleanOperator.OR)
            deployments.each {
                rule.addValues(
                        ResourceCollectionOuterClass.RuleValue.newBuilder().setValue(it)
                                .setMatchType(ResourceCollectionOuterClass.MatchType.EXACT)
                )
            }
            selector.addRules(rule)
        }

        if (namespaces.size() > 0) {
            def rule = ResourceCollectionOuterClass.SelectorRule.newBuilder()
                    .setFieldName("Namespace")
                    .setOperator(PolicyOuterClass.BooleanOperator.OR)
            namespaces.each {
                rule.addValues(
                        ResourceCollectionOuterClass.RuleValue.newBuilder().setValue(it)
                                .setMatchType(ResourceCollectionOuterClass.MatchType.EXACT)
                )
            }
            selector.addRules(rule)
        }

        def req = ResourceCollectionService.CreateCollectionRequest.newBuilder()
                .setName("Test Collections-${UUID.randomUUID()}")
                .addResourceSelectors(selector)
        return getClient().createCollection(req.build()).getCollection()
    }

    static deleteCollection(String collectionId) {
        getClient().deleteCollection(Common.ResourceByID.newBuilder().setId(collectionId).build())
    }
}
