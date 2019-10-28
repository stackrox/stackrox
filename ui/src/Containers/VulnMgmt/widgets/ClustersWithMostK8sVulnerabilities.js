import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';
import queryService from 'modules/queryService';

import workflowStateContext from 'Containers/workflowStateContext';

import ViewAllButton from 'Components/ViewAllButton';
import Loader from 'Components/Loader';
import Widget from 'Components/Widget';
import NumberedGrid from 'Components/NumberedGrid';
import FixableCVECount from 'Components/FixableCVECount';

import kubeSVG from 'images/kube.svg';

const CLUSTER_WITH_MOST_K8S_VULNERABILTIES = gql`
    query clustersWithMostK8sVulnerabilities($query: String) {
        results: clusters(query: $query) {
            id
            name
            vulns {
                id: cve
                cve
                isFixable
            }
        }
    }
`;

const processData = (data, workflowState) => {
    const results = data.results.map(({ id, name: text, vulns }) => {
        const cveCount = vulns.length;
        const fixableCount = vulns.filter(vuln => vuln.isFixable).length;
        const url = workflowState.pushRelatedEntity(entityTypes.CLUSTER, id).toURL();
        return {
            text,
            url,
            component: (
                <div className="flex flex-1 justify-left">
                    <img src={kubeSVG} alt="kube" className="pr-2" />
                    <FixableCVECount
                        cves={cveCount}
                        fixable={fixableCount}
                        orientation="vertical"
                    />
                </div>
            )
        };
    });
    return results.slice(0, 8); // @TODO: Remove and add pagination when available
};

const ClustersWithMostK8sVulnerabilities = ({ entityContext }) => {
    const { loading, data = {} } = useQuery(CLUSTER_WITH_MOST_K8S_VULNERABILTIES, {
        variables: {
            query: queryService.entityContextToQueryString(entityContext)
        }
    });

    let content = <Loader />;

    const workflowState = useContext(workflowStateContext);
    if (!loading) {
        const processedData = processData(data, workflowState);

        content = (
            <div className="w-full">
                <NumberedGrid data={processedData} />
            </div>
        );
    }

    const viewAllURL = workflowState.pushList(entityTypes.CLUSTER).toURL();

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
    entityContext: PropTypes.shape({})
};

ClustersWithMostK8sVulnerabilities.defaultProps = {
    entityContext: {}
};

export default ClustersWithMostK8sVulnerabilities;
