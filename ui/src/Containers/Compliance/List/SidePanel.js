import React from 'react';
import PropTypes from 'prop-types';
import Panel from 'Components/Panel';
import ControlPage from 'Containers/Compliance/Entity/Control';
import ReactRouterPropTypes from 'react-router-prop-types';
import { resourceTypes, standardEntityTypes } from 'constants/entityTypes';
import { Link, withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import { CLUSTER_NAME } from 'queries/cluster';
import { NODE_NAME } from 'queries/node';
import { NAMESPACE_NAME } from 'queries/namespace';
import { DEPLOYMENT_NAME } from 'queries/deployment';
import { CONTROL_QUERY } from 'queries/controls';
import Query from 'Components/ThrowingQuery';
import { standardLabels } from 'messages/standards';
import * as Icon from 'react-feather';
import NamespacePage from '../Entity/Namespace';
import ClusterPage from '../Entity/Cluster';
import NodePage from '../Entity/Node';
import DeploymentPage from '../Entity/Deployment';

const ComplianceListSidePanel = ({ entityType, entityId, match, location, history }) => {
    function getEntityPage() {
        switch (entityType) {
            case resourceTypes.NODE:
                return <NodePage entityId={entityId} sidePanelMode />;
            case resourceTypes.NAMESPACE:
                return <NamespacePage entityId={entityId} sidePanelMode />;
            case resourceTypes.CLUSTER:
                return <ClusterPage entityId={entityId} sidePanelMode />;
            case resourceTypes.DEPLOYMENT:
                return <DeploymentPage entityId={entityId} sidePanelMode />;
            case standardEntityTypes.CONTROL:
                return <ControlPage entityId={entityId} sidePanelMode />;
            default:
                return null;
        }
    }

    function getQuery() {
        switch (entityType) {
            case resourceTypes.NODE:
                return NODE_NAME;
            case resourceTypes.NAMESPACE:
                return NAMESPACE_NAME;
            case resourceTypes.CLUSTER:
                return CLUSTER_NAME;
            case resourceTypes.DEPLOYMENT:
                return DEPLOYMENT_NAME;
            case standardEntityTypes.CONTROL:
                return CONTROL_QUERY;
            default:
                return null;
        }
    }

    function getLinkText(data) {
        switch (entityType) {
            case resourceTypes.NAMESPACE:
                return data.result.metadata.name;
            case standardEntityTypes.CONTROL:
                return `${standardLabels[data.results.standardId]} : ${data.results.name}`;
            default:
                return data.result.name;
        }
    }

    function closeSidePanel() {
        const baseURL = URLService.getURL(match, location)
            .clearSidePanelParams()
            .url();
        history.push(baseURL);
    }
    const headerUrl = URLService.getURL(match, location)
        .base(entityType, entityId)
        .url();

    return (
        <Query query={getQuery()} variables={{ id: entityId }}>
            {({ loading, data }) => {
                let linkText = 'loading...';
                if (!loading && data) {
                    linkText = getLinkText(data);
                }
                const headerTextComponent = (
                    <div className="w-full flex items-center">
                        <div className="flex items-center">
                            <Link
                                to={headerUrl}
                                className="w-full flex text-primary-700 hover:text-primary-800 focus:text-primary-700"
                            >
                                <div
                                    className="flex flex-1 uppercase items-center tracking-wide pl-4 leading-normal font-700"
                                    data-test-id="panel-header"
                                >
                                    {linkText}
                                </div>
                            </Link>
                            <Link
                                className="mx-2 text-primary-700 hover:text-primary-800 p-1 bg-primary-300 rounded flex"
                                to={headerUrl}
                                target="_blank"
                            >
                                <Icon.ExternalLink size="14" />
                            </Link>
                        </div>
                    </div>
                );
                const entityPage = getEntityPage();
                return (
                    <Panel
                        className="bg-primary-200 z-40 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-108 md:relative"
                        headerTextComponent={headerTextComponent}
                        onClose={closeSidePanel}
                    >
                        {entityPage}
                    </Panel>
                );
            }}
        </Query>
    );
};

ComplianceListSidePanel.propTypes = {
    history: ReactRouterPropTypes.history.isRequired,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired
};

export default withRouter(ComplianceListSidePanel);
