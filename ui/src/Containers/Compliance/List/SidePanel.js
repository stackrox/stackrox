import React from 'react';
import PropTypes from 'prop-types';
import contexts from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import Panel from 'Components/Panel';
import ControlPage from 'Containers/Compliance/Entity/Control';

import AppLink from 'Components/AppLink';
import entityTypes, { resourceTypes } from 'constants/entityTypes';
import NamespacePage from '../Entity/Namespace';
import ClusterPage from '../Entity/Cluster';
import NodePage from '../Entity/Node';
import DeploymentPage from '../Entity/Deployment';

const ComplianceListSidePanel = ({
    entityType,
    entityId,
    clearSelectedRow,
    linkText,
    standardId,
    controlResult
}) => {
    const isControl = entityType === entityTypes.CONTROL;
    const linkParams = {
        entityId,
        entityType,
        standardId,
        controlId: entityId
    };

    function getEntityPage() {
        if (isControl)
            return <ControlPage controlId={entityId} controlResult={controlResult} sidePanelMode />;

        switch (entityType) {
            case resourceTypes.NODE:
                return <NodePage nodeId={entityId} sidePanelMode />;
            case resourceTypes.NAMESPACE:
                return <NamespacePage namespaceId={entityId} sidePanelMode />;
            case resourceTypes.CLUSTER:
                return <ClusterPage clusterId={entityId} sidePanelMode />;
            case resourceTypes.DEPLOYMENT:
                return <DeploymentPage deploymentId={entityId} sidePanelMode />;
            default:
                return null;
        }
    }
    const headerTextComponent = (
        <div className="w-full flex items-center">
            <div>
                <AppLink
                    context={contexts.COMPLIANCE}
                    externalLink
                    pageType={pageTypes.ENTITY}
                    entityType={entityType}
                    params={linkParams}
                    className="w-full flex text-primary-700 hover:text-primary-800 focus:text-primary-700"
                >
                    <div
                        className="flex flex-1 uppercase items-center tracking-wide pl-4 leading-normal font-700"
                        data-test-id="panel-header"
                    >
                        {linkText}
                    </div>
                </AppLink>
            </div>
        </div>
    );

    return (
        <Panel
            className="bg-primary-200 z-40 w-full h-full absolute pin-r pin-t md:w-1/2 min-w-108 md:relative"
            headerTextComponent={headerTextComponent}
            onClose={clearSelectedRow}
        >
            {getEntityPage()}
        </Panel>
    );
};

ComplianceListSidePanel.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    clearSelectedRow: PropTypes.func,
    linkText: PropTypes.string,
    standardId: PropTypes.string,
    controlResult: PropTypes.shape({})
};

ComplianceListSidePanel.defaultProps = {
    clearSelectedRow: () => {},
    linkText: 'Link',
    standardId: null,
    controlResult: null
};

export default ComplianceListSidePanel;
