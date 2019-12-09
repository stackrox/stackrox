import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';
import sortBy from 'lodash/sortBy';

import workflowStateContext from 'Containers/workflowStateContext';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedGrid from 'Components/NumberedGrid';
import FixableCVECount from 'Components/FixableCVECount';

import { HelpCircle, AlertCircle } from 'react-feather';

import kubeSVG from 'images/kube.svg';

import Tooltip from 'rc-tooltip';

// need to add query for fixable cves for dashboard once it's supported
const CLUSTER_WITH_MOST_K8S_VULNERABILTIES = gql`
    query clustersWithMostK8sVulnerabilities {
        results: clusters {
            id
            name
            isGKECluster
            k8sVulnCount
            k8sVulns {
                cve
                isFixable
            }
        }
    }
`;

const processData = (data, workflowState, limit) => {
    if (!data.results) return [];
    const stacked = data.results.length < 4;
    const results = sortBy(data.results, ['k8sVulnCount'])
        .slice(-limit)
        .map(({ id, name, isGKECluster, k8sVulns }) => {
            const cveCount = k8sVulns.length;
            const fixableCount = k8sVulns.filter(vuln => vuln.isFixable).length;
            const targetState = workflowState
                .pushRelatedEntity(entityTypes.CLUSTER, id)
                .pushList(entityTypes.CVE)
                .setSearch({ 'Vulnerability Type': 'K8S_VULNERABILITY' });

            const imgComponent = (
                <img src={kubeSVG} alt="kube" className={`${stacked ? 'pl-2' : 'pr-2'}`} />
            );

            const indicationTooltipText = isGKECluster
                ? 'These CVEs might have been patched by GKE. Please check the GKE release notes or security bulletin to find out more.'
                : 'These CVEs were not patched in the current Kubernetes version of this cluster';

            const indicatorIcon = isGKECluster ? (
                <HelpCircle className="h-4 w-4 text-warning-700 ml-2" />
            ) : (
                <AlertCircle className="h-4 w-4 text-alert-700 ml-2" />
            );

            const url = targetState.toUrl();
            const fixableUrl = targetState
                .setSearch({
                    'Fixed By': 'r/.*',
                    'Vulnerability Type': 'K8S_VULNERABILITY'
                })
                .toUrl();

            const content = (
                <div className="flex flex-1 items-center justify-left">
                    {!stacked && imgComponent}
                    <FixableCVECount
                        cves={cveCount}
                        url={url}
                        fixableUrl={fixableUrl}
                        fixable={fixableCount}
                        orientation={stacked ? 'horizontal' : 'vertical'}
                    />
                    {stacked && imgComponent}
                    <Tooltip placement="top" overlay={<div>{indicationTooltipText}</div>}>
                        {indicatorIcon}
                    </Tooltip>
                </div>
            );

            return {
                text: name,
                url,
                component: content
            };
        });
    return results.slice(0, 8); // @TODO: Remove and add pagination when available
};

const ClustersWithMostK8sVulnerabilities = ({ entityContext, limit }) => {
    const { loading, data = {} } = useQuery(CLUSTER_WITH_MOST_K8S_VULNERABILTIES, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext)
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState, limit);

        content = (
            <div className="w-full">
                <NumberedGrid data={processedData} />
            </div>
        );
    }

    const viewAllURL = workflowState
        .pushList(entityTypes.CLUSTER)
        .setSort([{ id: 'vulnCounter.all.total', desc: true }])
        .toUrl();

    return (
        <Widget
            className="h-full pdf-page"
            header="Clusters With Most K8s Vulnerabilities"
            headerComponents={<ViewAllButton url={viewAllURL} />}
        >
            {content}
        </Widget>
    );
};

ClustersWithMostK8sVulnerabilities.propTypes = {
    entityContext: PropTypes.shape({}),
    limit: PropTypes.number
};

ClustersWithMostK8sVulnerabilities.defaultProps = {
    entityContext: {},
    limit: 8
};

export default ClustersWithMostK8sVulnerabilities;
