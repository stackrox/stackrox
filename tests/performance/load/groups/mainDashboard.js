import { group } from 'k6';
import http from 'k6/http';

export function mainDashboard(host, headers, tags) {
    group('main dashboard', function () {
        http.post(
            `${host}/api/graphql?opname=summary_counts`,
            '{"operationName":"summary_counts","variables":{},"query":"query summary_counts {\\n  clusterCount\\n  nodeCount\\n  violationCount\\n  deploymentCount\\n  imageCount\\n  secretCount\\n}\\n"}',
            { headers, tags }
        );
        /** DB Queries
         *  - clusterCount => Cached store scan
         *  - nodeCount
         *  select count(*) from nodes where nodes.ClusterId = $1
         *  - violationCount ($12 = 0, $13 = 3)
         *  select count(*) from alerts where ((alerts.ClusterId = $1 and (alerts.Namespace = $2 or ... or alerts.Namespace = $11)) and ((alerts.State = $12) or (alerts.State = $13)))
         *  - deploymentCount => Cached store scan
         *  - imageCount
         *  select count(distinct(images.Id)) from images inner join deployments_containers on images.Id = deployments_containers.Image_Id inner join deployments on deployments_containers.deployments_Id = deployments.Id where (deployments.ClusterId = $1 and (deployments.Namespace = $2 or ... or deployments.Namespace = $11))
         *  - secretCount
         *  select count(*) from secrets where (secrets.ClusterId = $1 and (secrets.Namespace = $2 or ... or secrets.Namespace = $11))
         **/

        http.post(
            `${host}/api/graphql?opname=getAllNamespacesByCluster`,
            '{"operationName":"getAllNamespacesByCluster","variables":{"query":""},"query":"query getAllNamespacesByCluster($query: String) {\\n  clusters(query: $query) {\\n    id\\n    name\\n    namespaces {\\n      metadata {\\n        id\\n        name\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
            { headers, tags }
        );
        /** DB Queries
         *  - clusters
         *  select clusters.Id::text as Cluster_ID from clusters where clusters.Id = $1 order by clusters.Name asc
         *  $2 = q1.clusters.Id
         *  select "cluster_health_statuses".serialized from cluster_health_statuses inner join clusters on cluster_health_statuses.Id = clusters.Id where (clusters.Id = $1 and cluster_health_statuses.Id = ANY($2::uuid[]))
         *  - namespaces
         *  $12 = q1.clusters.Id
         *  select namespaces.Id::text as Namespace_ID from namespaces where ((namespaces.ClusterId = $1 and (namespaces.Name = $2 or ... or namespaces.Name = $11)) and namespaces.ClusterId = $12) order by namespaces.Name asc
         *  $12 = q3.namespaces.Id, $13 = q1.clusters.Id
         *  select namespaces.Id::text as Namespace_ID from namespaces where ((namespaces.ClusterId = $1 and (namespaces.Name = $2 or ... or namespaces.Name = $11)) and (namespaces.Id = ANY($12::uuid[]) and namespaces.ClusterId = $13)) order by namespaces.Name asc
         *
         *  Note: cluster and namespace payloads are retrieved from the cached stores and have no SQL footprint.
         **/

        http.post(
            `${host}/api/graphql?opname=alertCountsBySeverity`,
            '{"operationName":"alertCountsBySeverity","variables":{"lowQuery":"Severity:LOW_SEVERITY","medQuery":"Severity:MEDIUM_SEVERITY","highQuery":"Severity:HIGH_SEVERITY","critQuery":"Severity:CRITICAL_SEVERITY"},"query":"query alertCountsBySeverity($lowQuery: String, $medQuery: String, $highQuery: String, $critQuery: String) {\\n  LOW_SEVERITY: violationCount(query: $lowQuery)\\n  MEDIUM_SEVERITY: violationCount(query: $medQuery)\\n  HIGH_SEVERITY: violationCount(query: $highQuery)\\n  CRITICAL_SEVERITY: violationCount(query: $critQuery)\\n}\\n"}',
            { headers, tags }
        );
        /** DB Queries
         *  - violationCount ($12 = 1..4, $13 = 0 -ViolationState_ACTIVE-, $14 = 3 -ViolationState_ATTEMPTED-)
         *  select count(*) from alerts where ((alerts.ClusterId = $1 and (alerts.Namespace = $2 or ... or alerts.Namespace = $11)) and ((alerts.Policy_Severity = $12) and ((alerts.State = $13) or (alerts.State = $14))))
         */

        http.post(
            `${host}/api/graphql?opname=mostRecentAlerts`,
            '{"operationName":"mostRecentAlerts","variables":{"query":"Severity:CRITICAL_SEVERITY"},"query":"query mostRecentAlerts($query: String) {\\n  alerts: violations(\\n    query: $query\\n    pagination: {limit: 3, sortOption: {field: \\"Violation Time\\", reversed: true}}\\n  ) {\\n    id\\n    time\\n    deployment {\\n      name\\n      __typename\\n    }\\n    resource {\\n      resourceType\\n      name\\n      __typename\\n    }\\n    policy {\\n      name\\n      severity\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
            { headers, tags }
        );
        /** DB Queries
         *  - violations ($12 = 4, $13 = 0, $15 = 3)
         *  select "alerts".serialized from alerts where ((alerts.ClusterId = $1 and (alerts.Namespace = $2 or ... or alerts.Namespace = $11)) and ((alerts.Policy_Severity = $12) and ((alerts.State = $13) or (alerts.State = $14)))) order by alerts.Time desc LIMIT 3
         *  (3x $12 = q1.alerts.serialized)
         *  select "alerts".serialized from alerts where ((alerts.ClusterId = $1 and (alerts.Namespace = $2 or ... or alerts.Namespace = $11)) and alerts.Id = ANY($12::uuid[]))
         */

        http.post(
            `${host}/api/graphql?opname=getImagesAtMostRisk`,
            '{"operationName":"getImagesAtMostRisk","variables":{"query":""},"query":"query getImagesAtMostRisk($query: String) {\\n  images(\\n    query: $query\\n    pagination: {limit: 6, sortOption: {field: \\"Image Risk Priority\\", reversed: false}}\\n  ) {\\n    id\\n    name {\\n      remote\\n      fullName\\n      __typename\\n    }\\n    priority\\n    imageVulnerabilityCounter {\\n      important {\\n        total\\n        fixable\\n        __typename\\n      }\\n      critical {\\n        total\\n        fixable\\n        __typename\\n      }\\n      __typename\\n    }\\n    __typename\\n  }\\n}\\n"}',
            { headers, tags }
        );
        /** DB Queries
         *  - images
         *  select distinct(images.Id) as Image_Sha, images.RiskScore as image_risk_score from images inner join deployments_containers on images.Id = deployments_containers.Image_Id inner join deployments on deployments_containers.deployments_Id = deployments.Id where (deployments.ClusterId = $1 and (deployments.Namespace = $2 or … or deployments.Namespace = $11)) order by images.RiskScore desc LIMIT 6
         *  ($23..$26 = q1.images.Id)
         *  select "images".serialized from images inner join deployments_containers on images.Id = deployments_containers.Image_Id inner join deployments on deployments_containers.deployments_Id = deployments.Id where ((deployments.ClusterId = $1 and (deployments.Namespace = $2 or … or deployments.Namespace = $11)) and ((deployments.ClusterId = $12 and (deployments.Namespace = $13 or … or deployments.Namespace = $22)) and (images.Id = $23 or images.Id = $24 or images.Id = $25 or images.Id = $26 or images.Id = $27 or images.Id = $28)))
         *  - imageVulnerabilityCounter
         *  ($12 = false, $13 = true..false, $14 = q1.images.Id)
         *  select distinct(image_cves.Id) as CVE_ID from image_cves inner join image_component_cve_edges on image_cves.Id = image_component_cve_edges.ImageCveId inner join image_component_edges on image_component_cve_edges.ImageComponentId = image_component_edges.ImageComponentId inner join images on image_component_edges.ImageId = images.Id inner join deployments_containers on images.Id = deployments_containers.Image_Id inner join deployments on deployments_containers.deployments_Id = deployments.Id where ((deployments.ClusterId = $1 and (deployments.Namespace = $2 or … or deployments.Namespace = $11)) and ((image_cves.Snoozed = $12 and image_component_cve_edges.IsFixable = $13) and images.Id = $14))
         *  ($12 = q3.image_cves.Id)
         *  select "image_cves".serialized from image_cves inner join image_component_cve_edges on image_cves.Id = image_component_cve_edges.ImageCveId inner join image_component_edges on image_component_cve_edges.ImageComponentId = image_component_edges.ImageComponentId inner join deployments_containers on image_component_edges.ImageId = deployments_containers.Image_Id inner join deployments on deployments_containers.deployments_Id = deployments.Id where ((deployments.ClusterId = $1 and (deployments.Namespace = $2 or … or deployments.Namespace = $11)) and image_cves.Id = ANY($12::text[]))
         */

        // GET v1/deploymentswithprocessinfo?pagination.offset=0&pagination.limit=6&pagination.sortOption.field=Deployment%20Risk%20Priority&pagination.sortOption.reversed=false
        http.get(
            `${host}/v1/deploymentswithprocessinfo?pagination.offset=0&pagination.limit=6&pagination.sortOption.field=Deployment%20Risk%20Priority&pagination.sortOption.reversed=false`,
            { headers, tags }
        );
        /** DB Queries
         *  - 6 deployments in scope with highest risk
         *  select deployments.Id::text as Deployment_ID from deployments where (deployments.ClusterId = $$ and (deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$)) order by deployments.RiskScore desc LIMIT 6
         *  - associated process data
         *  select "process_baseline_results".serialized from process_baseline_results where process_baseline_results.DeploymentId = ANY($$::uuid[])
         */

        // GET v1/deployments?pagination.offset=0&pagination.limit=6&pagination.sortOption.field=Deployment%20Risk%20Priority&pagination.sortOption.reversed=false
        http.get(
            `${host}/v1/deployments?pagination.offset=0&pagination.limit=6&pagination.sortOption.field=Deployment%20Risk%20Priority&pagination.sortOption.reversed=false`,
            { headers, tags }
        );
        /** DB Queries
         *  - 6 deployments in scope with highest risk
         *  select deployments.Id::text as Deployment_ID from deployments where (deployments.ClusterId = $$ and (deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$ or deployments.Namespace = $$)) order by deployments.RiskScore desc LIMIT 6
         */

        http.post(
            `${host}/api/graphql?opname=agingImagesQuery`,
            '{"operationName":"agingImagesQuery","variables":{"query0":"Image Created Time:30d-90d","query1":"Image Created Time:90d-180d","query2":"Image Created Time:180d-365d","query3":"Image Created Time:>365d"},"query":"query agingImagesQuery($query0: String, $query1: String, $query2: String, $query3: String) {\\n  timeRange0: imageCount(query: $query0)\\n  timeRange1: imageCount(query: $query1)\\n  timeRange2: imageCount(query: $query2)\\n  timeRange3: imageCount(query: $query3)\\n}\\n"}',
            { headers, tags }
        );
        /** DB Queries
         *  - imageCount
         *  (query0..2 time ranges)
         *  select count(distinct(images.Id)) from images inner join deployments_containers on images.Id = deployments_containers.Image_Id inner join deployments on deployments_containers.deployments_Id = deployments.Id where ((deployments.ClusterId = $1 and (deployments.Namespace = $2 or ... or deployments.Namespace = $11)) and images.Metadata_V1_Created > $12 and images.Metadata_V1_Created < $13)
         *  (query3 older than)
         *  select count(distinct(images.Id)) from images inner join deployments_containers on images.Id = deployments_containers.Image_Id inner join deployments on deployments_containers.deployments_Id = deployments.Id where ((deployments.ClusterId = $1 and (deployments.Namespace = $2 or ... or deployments.Namespace = $11)) and images.Metadata_V1_Created <= $12)
         */

        // GET v1/alerts/summary/counts?request.query=&group_by=CATEGORY
        http.get(
            `${host}/v1/alerts/summary/counts?request.query=&group_by=CATEGORY`,
            { headers, tags }
        );
        /** DB Queries
         *
         *  select alerts.Id::text as Alert_ID, alerts.ClusterName, alerts.Policy_Severity, alerts.Policy_Categories from alerts where ((alerts.ClusterId = $$ and (alerts.Namespace = $$ or ... or alerts.Namespace = $$)) and (((alerts.ClusterName is not null and alerts.Policy_Severity is not null) and alerts.Policy_Categories is not null) and ((alerts.State = $$) or (alerts.State = $$))))
         */

        http.post(
            `${host}/api/graphql?opname=getAggregatedResults`,
            '{"operationName":"getAggregatedResults","variables":{"groupBy":["STANDARD"],"where":"Cluster:*"},"query":"query getAggregatedResults($groupBy: [ComplianceAggregation_Scope!], $where: String) {\\n  controls: aggregatedResults(groupBy: $groupBy, unit: CONTROL, where: $where) {\\n    results {\\n      aggregationKeys {\\n        id\\n        scope\\n        __typename\\n      }\\n      numFailing\\n      numPassing\\n      numSkipped\\n      unit\\n      __typename\\n    }\\n    __typename\\n  }\\n  complianceStandards: complianceStandards {\\n    id\\n    name\\n    __typename\\n  }\\n}\\n"}',
            { headers, tags }
        );
        /** DB Queries
         *  - complianceStandards ?
         *  select "compliance_configs".serialized from compliance_configs where compliance_configs.StandardId = ANY($1::text[])
         *
         *  There probably is some caching involved here as well.
         */
    });
}
