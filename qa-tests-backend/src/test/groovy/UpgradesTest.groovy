import com.google.protobuf.util.JsonFormat
import groovy.io.FileType
import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.api.v1.SummaryServiceOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ScopeOuterClass

import services.ClusterService
import services.GraphQLService
import services.PolicyService
import services.SummaryService
import util.Env

import spock.lang.Tag
import spock.lang.Unroll
import spock.lang.IgnoreIf

class UpgradesTest extends BaseSpecification {
    private final static String CLUSTERID = Env.mustGet("UPGRADE_CLUSTER_ID")
    private final static String POLICIES_JSON_PATH =
            Env.get("POLICIES_JSON_RELATIVE_PATH", "../pkg/defaults/policies/files")

    private static final String VULNERABILITY_RESOURCE_TYPE =
        isPostgresRun() ?
            "nodeVulnerabilities" :
            "vulnerabilities"

    private static final String COMPONENT_RESOURCE_TYPE =
        isPostgresRun() ?
            "nodeComponents" :
            "components"

    private static final COMPLIANCE_QUERY = """query getAggregatedResults(
        \$groupBy: [ComplianceAggregation_Scope!],
        \$unit: ComplianceAggregation_Scope!,
        \$where: String) {
            results: aggregatedResults(groupBy: \$groupBy, unit: \$unit, where: \$where) {
                results {
                    aggregationKeys {
                          id
                    }
                    unit
                }
            }
        }"""

    @Tag("Upgrade")
    @Tag("PZ")
    def "Verify cluster has listen on exec/pf webhook turned on"() {
        expect:
        "Migrated clusters to have admissionControllerEvents set to true"
        def cluster = ClusterService.getCluster()
        cluster != null
        assert(cluster.getAdmissionControllerEvents() == true)
    }

    @Tag("Upgrade")
    @Tag("PZ")
    def "Verify cluster has disable audit logs set to true"() {
        expect:
        "Migrated k8s clusters to have disableAuditLogs set to true"
        def cluster = ClusterService.getCluster()
        cluster != null
        assert(cluster.getDynamicConfig().getDisableAuditLogs() == true)
    }

    @Tag("Upgrade")
    @Tag("PZ")
    def "Verify that summary API returns non-zero values on upgrade"() {
        expect:
        "Summary API returns non-zero values on upgrade"
        SummaryServiceOuterClass.SummaryCountsResponse resp = SummaryService.getCounts()
        assert resp.numAlerts != 0
        assert resp.numDeployments != 0
        assert resp.numSecrets != 0
        assert resp.numClusters != 0
        assert resp.numImages != 0
        assert resp.numNodes != 0
    }

    @Unroll
    @Tag("Upgrade")
    @Tag("PZ")
    def "verify that we find the correct number of #resourceType for query"() {
        when:
        "Fetch the #resourceType from GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(getQuery(resourceType), [ query: searchQuery ])
        assert resultRet.getCode() == 200
        log.info "return code " + resultRet.getCode()

        then:
        "Check that we got the correct number of #resourceType from GraphQL "
        assert resultRet.getValue() != null
        def items = resultRet.getValue()[resourceType]
        assert items.size() >= minResults

        where:
        "Data Inputs Are:"
        resourceType      | searchQuery               | minResults
        "policies"        | "Policy:Latest Tag"       | 1
        "nodes"           | "Cluster ID:${CLUSTERID}" | 2
        "violations"      | ""                        | 1
        "secrets"         | "Cluster ID:${CLUSTERID}" | 1
        "deployments"     | "Cluster ID:${CLUSTERID}" | 1
        "images"          | "Cluster ID:${CLUSTERID}" | 1
        COMPONENT_RESOURCE_TYPE      | "Cluster ID:${CLUSTERID}" | 1
        VULNERABILITY_RESOURCE_TYPE | "Cluster ID:${CLUSTERID}" | 1
    }

    static getQuery(resourceType) {
        return """query get${resourceType}(\$query: String!) {
                ${resourceType} : ${resourceType}(query: \$query) {
                     id
                }
            }"""
    }

    @Unroll
    @Tag("Upgrade")
    @Tag("PZ")
    def "verify that we find the correct number of compliance results"() {
        when:
        "Fetch the compliance results by #unit from GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(COMPLIANCE_QUERY, [ groupBy: groupBy, unit: unit ])
        assert resultRet.getCode() == 200
        log.info "return code " + resultRet.getCode()

        then:
        "Check that we got the correct number of #unit from GraphQL "
        assert resultRet.getValue() != null
        def resultList = resultRet.getValue()["results"]
        assert resultList.size() >= numResults

        where:
        "Data Inputs Are:"
        groupBy                   | unit      | numResults
        ["STANDARD", "CLUSTER"]   | "CHECK"   | 1
        ["STANDARD", "NAMESPACE"] | "CHECK"   | 1
        ["STANDARD", "CLUSTER"]   | "CONTROL" | 1
        ["STANDARD", "NAMESPACE"] | "CONTROL" | 1
    }

    static private class KnownPolicyDiffs {
        Set<PolicyOuterClass.Exclusion> toRemove
        List<PolicyOuterClass.Exclusion> toAdd
        String clusterId = null
        Boolean setDisabled = null
        boolean clearEnforcement = false
        boolean clearLastUpdatedTs = false

        def addExclusionsWithName(def toAdd) {
            this.toAdd = toAdd.collect {
                def dep = PolicyOuterClass.Exclusion.Deployment.newBuilder().
                        setScope(ScopeOuterClass.Scope.newBuilder().setNamespace(it[0])).
                        setName(it[1]).
                        build()
                PolicyOuterClass.Exclusion.newBuilder().
                        setName("Don't alert on ${it[2]}").setDeployment(dep).build()
            }
            return this
        }

        def removeExclusions(def toRemove) {
            this.toRemove = toRemove.collect {
                def dep = PolicyOuterClass.Exclusion.Deployment.newBuilder().
                        setScope(ScopeOuterClass.Scope.newBuilder().setNamespace(it[0])).
                        setName(it[1]).
                        build()
                PolicyOuterClass.Exclusion.newBuilder().
                        setName("${it[2]}").setDeployment(dep).build()
            }
            return this
        }

        def applyToCluster(String id) {
            this.clusterId = id
            return this
        }

        def clearEnforcementActions() {
            this.clearEnforcement = true
            return this
        }

        def clearLastUpdated() {
            this.clearLastUpdatedTs = true
            return this
        }

        def setPolicyAsDisabled() {
            this.setDisabled = true
            return this
        }

        def setPolicyAsEnabled() {
            this.setDisabled = false
            return this
        }
    }

    @Tag("Upgrade")
    @Tag("PZ")
    @IgnoreIf({ true }) // ROX-16401 this test will not work with current upgrade methodology & image tags
    def "Verify upgraded policies match default policy set"() {
        given:
        "Default policies in code"

        def policiesGuardedByFeatureFlags = []
        Map<String, PolicyOuterClass.Policy> defaultPolicies = [:]
        def policiesDir = new File(POLICIES_JSON_PATH)
        policiesDir.eachFileRecurse (FileType.FILES) { file ->
            if (file.name.endsWith(".json")) {
                def builder = PolicyOuterClass.Policy.newBuilder()
                JsonFormat.parser().merge(file.text, builder)
                def policy = builder.build()
                if (!policiesGuardedByFeatureFlags.contains(policy.id)) {
                    defaultPolicies[policy.id] = policy
                }
            }
        }

        when:
        "Upgraded default policies are fetched from central"
        def upgradedPolicies
        try {
            log.info("Exporting policies: ${defaultPolicies.keySet().join(", ")}")
            upgradedPolicies = PolicyService.getPolicyClient().exportPolicies(
                    PolicyServiceOuterClass.ExportPoliciesRequest.newBuilder().
                            addAllPolicyIds(defaultPolicies.keySet()).
                            build()
            ).getPoliciesList()
        } catch (StatusRuntimeException e) {
            log.info "Exception in exportPolicies(): ${e.getStatus()}"
            log.info "See central log for more details."
            throw(e)
        }

        def knownPolicyDifferences = [
            "2e90874a-3521-44de-85c6-5720f519a701" : new KnownPolicyDiffs()
            // this diff is only for the 56.1 upgrade test
                .applyToCluster("268c98c6-e983-4f4e-95d2-9793cebddfd7")
                .removeExclusions([
                        ["kube-system", "", ""],
                        ["istio-system", "", ""]
                ])
                .addExclusionsWithName([
                        ["kube-system", "", "kube-system namespace", 0],
                        ["istio-system", "", "istio-system namespace", 1]
                ])
                .clearEnforcementActions()
                .clearLastUpdated(),
            "1913283f-ce3c-4134-84ef-195c4cd687ae" : new KnownPolicyDiffs().setPolicyAsDisabled(),
            "842feb9f-ecb1-4e3c-a4bf-8a1dcb63948a" : new KnownPolicyDiffs().setPolicyAsDisabled(),
            "f09f8da1-6111-4ca0-8f49-294a76c65115" : new KnownPolicyDiffs().setPolicyAsDisabled(),
            "a919ccaf-6b43-4160-ac5d-a405e1440a41" : new KnownPolicyDiffs().setPolicyAsEnabled(),
            "93f4b2dd-ef5a-419e-8371-38aed480fb36" : new KnownPolicyDiffs().setPolicyAsDisabled(),
        ]
        and:
        "Skip over known differences due to differences in tests"
        upgradedPolicies = upgradedPolicies.collect { policy ->
            assert Float.parseFloat(policy.policyVersion) >= 1.0

            def builder = PolicyOuterClass.Policy.newBuilder(policy)

            // All default policies are expected to have the following flags set to true.
            // Therefore, move to target by adding the known diff.
            builder.setCriteriaLocked(true)
            // The upgrade tests upgrades policies from prior to 65.0, which we do not migrate
            // because of above - criteria is not locked.
            builder.setIsDefault(true)
            if (knownPolicyDifferences.containsKey(policy.id)) {
                def diffs = knownPolicyDifferences[policy.id]

                if (diffs.clusterId == null || diffs.clusterId == CLUSTERID) {
                    if (diffs.toRemove) {
                        def filteredExclusions = policy.exclusionsList.findAll { !diffs.toRemove.contains(it) }
                        builder.clearExclusions().addAllExclusions(filteredExclusions)
                    }
                    if (diffs.toAdd) {
                        diffs.toAdd.each { builder.addExclusions(it) }
                    }
                    if (diffs.clearLastUpdatedTs) {
                        builder.clearLastUpdated()
                    }
                    if (diffs.clearEnforcement) {
                        builder.clearEnforcementActions()
                    }
                }
                if (diffs.setDisabled != null) {
                    builder.setDisabled(diffs.setDisabled.booleanValue())
                }
            }

            builder.build()
        }

        and:
        "Ignore ordering for exclusions and categories in policies by resorting them"
        upgradedPolicies = upgradedPolicies.collect { policy ->
            def builder = PolicyOuterClass.Policy.newBuilder(policy)
            if (policy.exclusionsList != null || !policy.exclusionsList.isEmpty()) {
                builder.clearExclusions().addAllExclusions(
                        // exclusionList is immutable, but .sort sees it as a list and assumes it's mutable
                        // so force it to not mutate otherwise this will throw
                        policy.exclusionsList.sort(false) { it.name }
                )
            }
            builder.clearCategories().addAllCategories(policy.categoriesList.sort(false))
            builder.build()
        }

        defaultPolicies = defaultPolicies.collectEntries { id, policy ->
            def builder = PolicyOuterClass.Policy.newBuilder(policy)

            if (policy.exclusionsList != null || !policy.exclusionsList.isEmpty()) {
                builder.clearExclusions().addAllExclusions(
                        policy.exclusionsList.sort(false) { it.name }
                )
            }
            builder.clearCategories().addAllCategories(policy.categoriesList.sort(false))
            [id, builder.build()]
        } as Map<String, PolicyOuterClass.Policy>

        then:
        "All default policies must still exist"
        assert upgradedPolicies.size() >= defaultPolicies.size()

        and:
        "Upgraded policies should match the default policies in code"
        upgradedPolicies.forEach {
            def defaultPolicy = defaultPolicies[it.id]
            assert it == defaultPolicy
        }
    }
}
