import com.google.protobuf.Timestamp
import com.google.protobuf.UnknownFieldSet
import com.google.protobuf.util.JsonFormat
import groovy.io.FileType
import groups.Upgrade
import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.api.v1.SummaryServiceOuterClass
import io.stackrox.proto.storage.ClusterOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ProcessIndicatorOuterClass
import io.stackrox.proto.storage.ScopeOuterClass
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ConfigService
import services.GraphQLService
import services.PolicyService
import services.ProcessService
import services.SummaryService
import spock.lang.Unroll
import util.Env

class UpgradesTest extends BaseSpecification {
    private final static String CLUSTERID = Env.mustGet("UPGRADE_CLUSTER_ID")
    private final static String POLICIES_JSON_PATH = Env.get("POLICIES_JSON_RELATIVE_PATH", "../image/policies/files")

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

    @Category(Upgrade)
    def "Verify cluster exists and that field values are retained"() {
        given:
        "Only run on specific upgrade from 2.4.16"
        Assume.assumeTrue(CLUSTERID=="260e11a3-cbea-464c-95f0-588fa7695b49")

        expect:
        def clusters = ClusterService.getClusters()
        clusters.size() == 1
        def expectedCluster = ClusterOuterClass.Cluster.newBuilder()
                .setId(CLUSTERID)
                .setName("remote")
                .setType(ClusterOuterClass.ClusterType.KUBERNETES_CLUSTER)
                .setPriority(1)
                .setMainImage("stackrox/main:2.4.16.4")
                .setCentralApiEndpoint("central.stackrox:443")
                .setCollectionMethod(ClusterOuterClass.CollectionMethod.KERNEL_MODULE)
                .setRuntimeSupport(true)
                .setTolerationsConfig(ClusterOuterClass.TolerationsConfig.newBuilder()
                        .setDisabled(true)
                        .build())
                .setHealthStatus(ClusterOuterClass.ClusterHealthStatus.newBuilder()
                        .setSensorHealthStatus(ClusterOuterClass.ClusterHealthStatus.HealthStatusLabel.UNHEALTHY)
                        .setCollectorHealthStatus(ClusterOuterClass.ClusterHealthStatus.HealthStatusLabel.UNAVAILABLE)
                        .setOverallHealthStatus(ClusterOuterClass.ClusterHealthStatus.HealthStatusLabel.UNHEALTHY)
                        .setLastContact(Timestamp.newBuilder().setSeconds(1551412107).setNanos(857477786).build())
                        .build()
                )
                .setStatus(ClusterOuterClass.ClusterStatus.newBuilder()
                        .setProviderMetadata(ClusterOuterClass.ProviderMetadata.newBuilder()
                                .setGoogle(ClusterOuterClass.GoogleProviderMetadata.newBuilder()
                                        .setProject("ultra-current-825")
                                        .setClusterName("setup-devde6c6")
                                        .build())
                                .setRegion("us-west1")
                                .setZone("us-west1-c")
                                .build())
                        .setOrchestratorMetadata(ClusterOuterClass.OrchestratorMetadata.newBuilder()
                                .setVersion("v1.11.7-gke.4")
                                .setBuildDate(Timestamp.newBuilder().setSeconds(1549394549).build())
                                .build())
                        .build())
                .setDynamicConfig(ClusterOuterClass.DynamicClusterConfig.newBuilder()
                        .setAdmissionControllerConfig(ClusterOuterClass.AdmissionControllerConfig.newBuilder()
                                .setTimeoutSeconds(3)))
                .build()

        def cluster = ClusterOuterClass.Cluster.newBuilder(clusters.get(0))
                .setUnknownFields(UnknownFieldSet.defaultInstance)
                .build()
        cluster == expectedCluster
    }

    @Category(Upgrade)
    def "Verify process indicators have cluster IDs and namespaces added"() {
        given:
        "Only run on specific upgrade from 2.4.16"
        Assume.assumeTrue(CLUSTERID=="260e11a3-cbea-464c-95f0-588fa7695b49")

        expect:
        "Migrated ProcessIndicators to have a cluster ID and a namespace"
        def processIndicators = ProcessService.getProcessIndicatorsByDeployment("33b3eb66-3bd4-11e9-b563-42010a8a0101")
        processIndicators.size() > 0
        for (ProcessIndicatorOuterClass.ProcessIndicator indicator : processIndicators) {
            assert(indicator.getClusterId() == CLUSTERID)
            assert(indicator.getNamespace() != "")
        }
    }

    @Category(Upgrade)
    def "Verify cluster has listen on exec/pf webhook turned on"() {
        given:
        Assume.assumeTrue(CLUSTERID=="260e11a3-cbea-464c-95f0-588fa7695b49")

        expect:
        "Migrated clusters to have admissionControllerEvents set to true"
        def cluster = ClusterService.getCluster()
        cluster != null
        assert(cluster.ClusterOuterClass.getAdmissionControllerEvents() == true)
    }

    @Category(Upgrade)
    def "Verify private config contains the correct retention duration for alerts and images"() {
        given:
        "Only run on specific upgrade from 2.4.16"
        Assume.assumeTrue(CLUSTERID=="260e11a3-cbea-464c-95f0-588fa7695b49")

        expect:
        "Alert retention duration is nil, image retention duration is 7 days"
        def config = ConfigService.getConfig()
        config != null
        config.getPrivateConfig().getAlertConfig() != null
        config.getPrivateConfig().getAlertConfig().getAllRuntimeRetentionDurationDays() == 0
        config.getPrivateConfig().getAlertConfig().getResolvedDeployRetentionDurationDays() == 0
        config.getPrivateConfig().getAlertConfig().getDeletedRuntimeRetentionDurationDays() == 0
        config.getPrivateConfig().getImageRetentionDurationDays() == 7
    }

    @Category(Upgrade)
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
    @Category(Upgrade)
    def "verify that we find the correct number of #resourceType for query"() {
        when:
        "Fetch the #resourceType from GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(getQuery(resourceType), [ query: searchQuery ])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()

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
        "components"      | "Cluster ID:${CLUSTERID}" | 1
        "vulnerabilities" | "Cluster ID:${CLUSTERID}" | 1
    }

    static getQuery(resourceType) {
        return """query get${resourceType}(\$query: String!) {
                ${resourceType} : ${resourceType}(query: \$query) {
                     id
                }
            }"""
    }

    @Unroll
    @Category(Upgrade)
    def "verify that we find the correct number of compliance results"() {
        when:
        "Fetch the compliance results by #unit from GraphQL"
        def gqlService = new GraphQLService()
        def resultRet = gqlService.Call(COMPLIANCE_QUERY, [ groupBy: groupBy, unit: unit ])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()

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
        List<Tuple2<PolicyOuterClass.Exclusion, Integer>> toAdd
        String remediation
        String rationale
        String description
        boolean setDisabled = false

        def addExclusions(def toAdd) {
            this.toAdd = toAdd.collect {
                def dep = PolicyOuterClass.Exclusion.Deployment.newBuilder().setScope(
                        ScopeOuterClass.Scope.newBuilder().setNamespace(it[0])
                )
                new Tuple2(
                        PolicyOuterClass.Exclusion.newBuilder().
                                setName("Don't alert on ${it[0]} namespace").setDeployment(dep).build(),
                        it[1]
                )
            }
            return this
        }

        def addExclusionsWithName(def toAdd) {
            this.toAdd = toAdd.collect {
                def dep = PolicyOuterClass.Exclusion.Deployment.newBuilder().
                        setScope(ScopeOuterClass.Scope.newBuilder().setNamespace(it[0])).
                        setName(it[1]).
                        build()
                new Tuple2(
                        PolicyOuterClass.Exclusion.newBuilder().
                                setName("Don't alert on ${it[2]}").setDeployment(dep).build(),
                        it[3]
                )
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
                        setName("Don't alert on ${it[2]}").setDeployment(dep).build()
            }
            return this
        }

        def updateRemediation(def remediation) {
            this.remediation = remediation
            return this
        }

        def updateRationale(def rationale) {
            this.rationale = rationale
            return this
        }

        def updateDescription(def description) {
            this.description = description
            return this
        }

        def setPolicyAsDisabled() {
            this.setDisabled = true
            return this
        }
    }

    @Category(Upgrade)
    def "Verify upgraded policies match default policy set"() {
        given:
        "Default policies in code"
        // TODO(nakul) update caveats once fixes are in place
        Assume.assumeFalse(CLUSTERID=="268c98c6-e983-4f4e-95d2-9793cebddfd7")

        def defaultPolicies = [:]
        def policiesDir = new File(POLICIES_JSON_PATH)
        policiesDir.eachFileRecurse (FileType.FILES) { file ->
            if (file.name.endsWith(".json")) {
                def builder = PolicyOuterClass.Policy.newBuilder()
                JsonFormat.parser().merge(file.text, builder)
                def policy = builder.build()

                defaultPolicies[policy.id] = policy
            }
        }

        when:
        "Upgraded default policies are fetched from central"
        def upgradedPolicies = PolicyService.getPolicyClient().exportPolicies(
                PolicyServiceOuterClass.ExportPoliciesRequest.newBuilder().
                        addAllPolicyIds(defaultPolicies.keySet()).
                        build()
        ).getPoliciesList()

        def knownPolicyDifferences = [
                "2db9a279-2aec-4618-a85d-7f1bdf4911b1": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "2e90874a-3521-44de-85c6-5720f519a701": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "886c3c94-3a6a-4f2b-82fc-d6bf5a310840": new KnownPolicyDiffs().addExclusions([["istio-system", 3]]),
                "fe9de18b-86db-44d5-a7c4-74173ccffe2e": new KnownPolicyDiffs().addExclusions([["istio-system", 2]]),
                "014a03c6-9053-49b5-88ea-c1efcf19804f": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "880fd131-46f0-43d2-82c9-547f5aa7e043": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "550081a1-ad3a-4eab-a874-8eb68fab2bbd": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "8ac93556-4ad4-4220-a275-3f518db0ceb9": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "d3e480c1-c6de-4cd2-9006-9a3eb3ad36b6": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "1a498d97-0cc2-45f5-b32e-1f3cca6a3113": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "60e7c7f3-dc78-4367-9e9a-68aa3b7467f0": new KnownPolicyDiffs().addExclusions([["istio-system", 1]]),
                "a9b9ecf7-9707-4e32-8b62-d03018ed454f": new KnownPolicyDiffs()
                        .removeExclusions([["kube-system", "", "kube namespace"]])
                        .addExclusionsWithName([
                                ["kube-system", "", "kube-system namespace", 2],
                                ["istio-system", "", "istio-system namespace", 3]
                        ])
                        .updateRemediation("Ensure that deployments do not mount sensitive host directories," +
                                " or exclude this deployment if host mount is required."),
                "7760a5f3-bca4-4ca8-94a7-ad89edbc0e2c": new KnownPolicyDiffs()
                        .removeExclusions([["kube-system", "", "Kube System Namespace"]])
                        .addExclusionsWithName([
                                ["kube-system", "", "kube-system namespace", 0],
                                ["istio-system", "", "istio-system namespace", 1]
                        ]),
                "f4996314-c3d7-4553-803b-b24ce7febe48": new KnownPolicyDiffs()
                        .removeExclusions([["stackrox", "scanner-v2-db", "StackRox scanner-v2 database"]])
                        .updateRemediation("Migrate your secrets from environment variables to orchestrator secrets" +
                                " or your security team's secret management solution."),
                "a788556c-9268-4f30-a114-d456f2380818": new KnownPolicyDiffs()
                        .removeExclusions([["stackrox", "scanner-v2-db", "StackRox scanner-v2 database"]])
                        .updateRemediation("Migrate your secrets from environment variables" +
                                " to your security team's secret management solution."),
                "74cfb824-2e65-46b7-b1b4-ba897e53af1f": new KnownPolicyDiffs()
                        .removeExclusions([
                                ["stackrox", "scanner-v2", "StackRox scanner-v2"],
                                ["stackrox", "scanner-v2-db", "StackRox scanner-v2 database"]
                        ])
                        .updateRemediation("Run `dpkg -r --force-all apt apt-get && dpkg -r --force-all debconf dpkg`" +
                                " in the image build for production containers."),
                "1913283f-ce3c-4134-84ef-195c4cd687ae": new KnownPolicyDiffs()
                        .removeExclusions([["stackrox", "scanner-v2", "StackRox scanner-v2"]])
                        .setPolicyAsDisabled(),
                "f95ff08d-130a-465a-a27e-32ed1fb05555": new KnownPolicyDiffs()
                        .removeExclusions([["stackrox", "scanner-v2", "StackRox scanner-v2"]])
                        .addExclusionsWithName([["stackrox", "scanner", "StackRox scanner", 0]]),
                "ddb7af9c-5ec1-45e1-a0cf-c36e3ef2b2ce": new KnownPolicyDiffs()
                        .removeExclusions([["stackrox", "scanner-v2", "StackRox scanner-v2"]])
                        .addExclusionsWithName([["stackrox", "scanner", "StackRox scanner", 0]]),
                "d7a275e1-1bba-47e7-92a1-42340c759883": new KnownPolicyDiffs().updateRemediation(
                        "Run `dpkg -r --force-all apt && dpkg -r --force-all debconf dpkg` in the image build" +
                                " for production containers. Change applications to no longer use package managers" +
                                " at runtime, if applicable."
                ),
                "89cae2e6-0cb7-4329-8692-c2c3717c1237": new KnownPolicyDiffs()
                        .updateRationale("A locked process baseline communicates high confidence that execution" +
                                " of a process not included in the baseline positively indicates malicious activity.")
                        .updateDescription("This policy generates a violation for any process execution" +
                                " that is not explicitly allowed by a locked process baseline" +
                                " for a given container specification within a Kubernetes deployment."),
                "842feb9f-ecb1-4e3c-a4bf-8a1dcb63948a": new KnownPolicyDiffs().setPolicyAsDisabled(),
        ]
        and:
        "Skip over known differences in migrated policies until ROX-6806 is fixed"
        upgradedPolicies = upgradedPolicies.collect { policy ->
            assert Float.parseFloat(policy.policyVersion) >= 1.0

            def builder = PolicyOuterClass.Policy.newBuilder(policy)
            if (policy.hasFields()) {
                builder.clearFields() // fields is ignored so clear it out
            }

            if (knownPolicyDifferences.containsKey(policy.id)) {
                def diffs = knownPolicyDifferences[policy.id]
                if (diffs.toRemove) {
                    def filteredExclusions = policy.exclusionsList.findAll { !diffs.toRemove.contains(it) }
                    builder.clearExclusions().addAllExclusions(filteredExclusions)
                }
                if (diffs.toAdd) {
                    diffs.toAdd.each { builder.addExclusions(it.second, it.first) }
                }
                if (diffs.remediation) {
                    builder.setRemediation(diffs.remediation)
                }
                if (diffs.rationale) {
                    builder.setRationale(diffs.rationale)
                }
                if (diffs.description) {
                    builder.setDescription(diffs.description)
                }
                if (diffs.setDisabled) {
                    builder.setDisabled(true)
                }
            }

            builder.build()
        }

        then:
        "All default policies must still exist"
        assert upgradedPolicies.size() >= defaultPolicies.size()

        and:
        "Upgraded policies should match the default policies in code"
        def unmatchedPolicies = upgradedPolicies.findAll {
            it != defaultPolicies[it.id]
        }
        assert unmatchedPolicies.size() == 0
    }
}
