import React from 'react';
import { withRouter, Switch, Route } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import URLService from 'modules/URLService';
import ClusterPage from './Cluster';
import NodePage from './Node';
import NamespacePage from './Namespace';

const ComplianceEntityPage = ({ match, location, params, sidePanelMode }) => {
    const pageParams = URLService.getParams(match, location);
    const pageProps = {
        params: Object.assign({}, pageParams, params),
        sidePanelMode
    };
    const ClusterEntityPage = () => <ClusterPage {...pageProps} />;
    const NodeEntityPage = () => <NodePage {...pageProps} />;
    const NamespaceEntityPage = () => <NamespacePage {...pageProps} />;
    /* eslint-disable */
    return (
        <Switch>
            <Route path="/main/compliance2/clusters" render={ClusterEntityPage} />
            <Route path="/main/compliance2/nodes" render={NodeEntityPage} />
            <Route path="/main/compliance2/namespaces" render={NamespaceEntityPage} />
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
