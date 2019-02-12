import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import { withRouter, Switch, Route } from 'react-router-dom';
import URLService from 'modules/URLService';
import contextTypes from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import entityTypes from 'constants/entityTypes';

import NodePage from './Node';
import NamespacePage from './Namespace';
import ClusterPage from './Cluster';
import ControlPage from './Control';

const ComplianceEntityPage = ({ match, location, params, sidePanelMode }) => {
    const pageParams = URLService.getParams(match, location);
    const pageProps = {
        params: Object.assign({}, pageParams, params),
        sidePanelMode
    };
    const ClusterEntityPage = () => <ClusterPage {...pageProps} />;
    const NodeEntityPage = () => <NodePage {...pageProps} />;
    const NamespaceEntityPage = () => <NamespacePage {...pageProps} />;
    const ControlEntityPage = () => <ControlPage {...pageProps} />;

    const clusterLink = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
        entityType: entityTypes.CLUSTER
    });
    const namespaceLink = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
        entityType: entityTypes.NAMESPACE
    });
    const nodeLink = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
        entityType: entityTypes.NODE
    });
    const controlLink = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
        entityType: pageParams.entityType
    });

    /* eslint-disable */
    return (
        <Switch>
            <Route path={clusterLink.url} render={ClusterEntityPage} />
            <Route path={namespaceLink.url} render={NamespaceEntityPage} />
            <Route path={nodeLink.url} render={NodeEntityPage} />
            <Route path={controlLink.url} render={ControlEntityPage} />
        </Switch>
    );
    /* eslint-enable */
};

ComplianceEntityPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    params: PropTypes.shape({
        entityId: PropTypes.string,
        entityType: PropTypes.string
    }),
    sidePanelMode: PropTypes.bool
};

ComplianceEntityPage.defaultProps = {
    params: null,
    sidePanelMode: false
};

export default withRouter(ComplianceEntityPage);
