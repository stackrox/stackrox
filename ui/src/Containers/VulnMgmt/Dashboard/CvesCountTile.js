import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { useQuery } from 'react-apollo';
import gql from 'graphql-tag';

import workflowStateContext from 'Containers/workflowStateContext';

import EntityTileLink from 'Components/EntityTileLink';

const CVES_COUNT_QUERY = gql`
    query cvesCount {
        vulnerabilities {
            cve
            isFixable
        }
    }
`;

const getURL = workflowState => {
    const url = workflowState.pushList(entityTypes.CVE).toUrl();
    return url;
};

const CvesCountTile = () => {
    const { loading, data = {} } = useQuery(CVES_COUNT_QUERY);

    const { vulnerabilities = [] } = data;

    const cveCount = vulnerabilities.length;
    const fixableCveCount = vulnerabilities.filter(vuln => !!vuln.isFixable).length;
    const fixableCveCountText = `(${fixableCveCount} fixable)`;

    const workflowState = useContext(workflowStateContext);
    const url = getURL(workflowState);

    return (
        <EntityTileLink
            count={cveCount}
            entityType={entityTypes.CVE}
            position="middle"
            subText={fixableCveCountText}
            loading={loading}
            isError={!!fixableCveCount}
            url={url}
        />
    );
};

export default CvesCountTile;
