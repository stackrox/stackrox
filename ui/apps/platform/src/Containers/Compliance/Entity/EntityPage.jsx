import React from 'react';
import PropTypes from 'prop-types';
import { useLocation } from 'react-router-dom';
import URLService from 'utils/URLService';
import useWorkflowMatch from 'hooks/useWorkflowMatch';

import ComplianceEntityCluster from './ComplianceEntityCluster';
import ComplianceEntityControl from './ComplianceEntityControl';
import ComplianceEntityDeployment from './ComplianceEntityDeployment';
import ComplianceEntityNamespace from './ComplianceEntityNamespace';
import ComplianceEntityNode from './ComplianceEntityNode';
import ComplianceEntityStandard from './ComplianceEntityStandard';

const EntityPage = () => {
    const location = useLocation();
    const match = useWorkflowMatch();

    const params = URLService.getParams(match, location);

    const pageProps = {
        entityId: params.pageEntityId,
        listEntityType1: params.entityListType1,
        entityType1: params.entityType1,
        entityId1: params.entityId1,
        entityType2: params.entityType2,
        entityListType2: params.entityListType2,
        entityId2: params.entityId2,
        query: params.query,
    };

    const pageTypeMap = {
        CLUSTER: <ComplianceEntityCluster {...pageProps} />,
        CONTROL: <ComplianceEntityControl {...pageProps} />,
        DEPLOYMENT: <ComplianceEntityDeployment {...pageProps} />,
        NAMESPACE: <ComplianceEntityNamespace {...pageProps} />,
        NODE: <ComplianceEntityNode {...pageProps} />,
        STANDARD: <ComplianceEntityStandard {...pageProps} />,
    };

    return pageTypeMap[params.pageEntityType];
};

EntityPage.propTypes = {
    params: PropTypes.shape({
        entityId: PropTypes.string,
        entityType: PropTypes.string,
    }),
    sidePanelMode: PropTypes.bool,
};

EntityPage.defaultProps = {
    params: null,
    sidePanelMode: false,
};

export default EntityPage;
