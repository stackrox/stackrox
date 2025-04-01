import React from 'react';
import PropTypes from 'prop-types';
import { useLocation } from 'react-router-dom';
import URLService from 'utils/URLService';
import entityTypes from 'constants/entityTypes';
import useWorkflowMatch from 'hooks/useWorkflowMatch';

import NodePage from './Node';
import NamespacePage from './Namespace';
import ClusterPage from './Cluster';
import ControlPage from './Control';
import DeploymentPage from './Deployment';
import StandardPage from './Standard';

const ComplianceEntityPage = () => {
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
        [entityTypes.CLUSTER]: <ClusterPage {...pageProps} />,
        [entityTypes.NODE]: <NodePage {...pageProps} />,
        [entityTypes.NAMESPACE]: <NamespacePage {...pageProps} />,
        [entityTypes.CONTROL]: <ControlPage {...pageProps} />,
        [entityTypes.DEPLOYMENT]: <DeploymentPage {...pageProps} />,
        [entityTypes.STANDARD]: <StandardPage {...pageProps} />,
    };

    return pageTypeMap[params.pageEntityType];
};

ComplianceEntityPage.propTypes = {
    params: PropTypes.shape({
        entityId: PropTypes.string,
        entityType: PropTypes.string,
    }),
    sidePanelMode: PropTypes.bool,
};

ComplianceEntityPage.defaultProps = {
    params: null,
    sidePanelMode: false,
};

export default ComplianceEntityPage;
