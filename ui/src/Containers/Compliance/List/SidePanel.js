import React from 'react';
import PropTypes from 'prop-types';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import Panel from 'Components/Panel';
import ControlPage from 'Containers/Compliance/Entity/Control';

import AppLink from 'Components/AppLink';
import entityTypes, {
    standardBaseTypes,
    standardTypes,
    resourceTypes
} from 'constants/entityTypes';
import NamespacePage from '../Entity/Namespace';
import ClusterPage from '../Entity/Cluster';
import NodePage from '../Entity/Node';

const ComplianceListSidePanel = ({ match, location, selectedRow, clearSelectedRow }) => {
    const { name, control } = selectedRow;
    const { context, query, entityType } = URLService.getParams(match, location);
    const { groupBy, ...rest } = query;
    const isControl = !!standardTypes[entityType];

    const linkParams = {
        query: rest,
        entityId: selectedRow.id,
        entityType
    };

    if (isControl) {
        linkParams.standardId = entityType;
        linkParams.controlId = selectedRow.id;
        linkParams.entityType = entityTypes.CONTROL;
    }

    function getEntityPage() {
        if (isControl) return <ControlPage controlId={selectedRow.id} sidePanelMode />;

        const entityId = selectedRow.id;
        switch (entityType) {
            case resourceTypes.NODE:
                return <NodePage nodeId={entityId} sidePanelMode />;
            case resourceTypes.NAMESPACE:
                return <NamespacePage namespaceId={entityId} sidePanelMode />;
            case resourceTypes.CLUSTER:
                return <ClusterPage clusterId={entityId} sidePanelMode />;
            default:
                return null;
        }
    }
    const headerTextComponent = (
        <div className="w-full flex items-center">
            <div>
                <AppLink
                    context={context}
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
                        {standardBaseTypes[entityType]
                            ? `${standardBaseTypes[entityType]} ${control}`
                            : name}
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
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    selectedRow: PropTypes.shape({}),
    clearSelectedRow: PropTypes.func.isRequired
};

ComplianceListSidePanel.defaultProps = {
    selectedRow: null
};

export default ComplianceListSidePanel;
