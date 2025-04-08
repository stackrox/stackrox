import React from 'react';
import PropTypes from 'prop-types';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import { Link, useLocation, useNavigate } from 'react-router-dom';

import Query from 'Components/CacheFirstQuery';
import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
import { resourceTypes, standardEntityTypes } from 'constants/entityTypes';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable import/no-cycle */
import Control from 'Containers/Compliance/Entity/Control';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import URLService from 'utils/URLService';
import getEntityName from 'utils/getEntityName';
import { entityNameQueryMap } from 'utils/queryMap';
import { truncate } from 'utils/textUtils';
import Namespace from '../Entity/Namespace';
import Cluster from '../Entity/Cluster';
import Node from '../Entity/Node';
import Deployment from '../Entity/Deployment';

const MAX_CONTROL_TITLE = 120;

const SidePanel = ({ entityType, entityId }) => {
    const navigate = useNavigate();
    const location = useLocation();
    const match = useWorkflowMatch();

    function getEntityPage() {
        switch (entityType) {
            case resourceTypes.NODE:
                return <Node entityId={entityId} sidePanelMode />;
            case resourceTypes.NAMESPACE:
                return <Namespace entityId={entityId} sidePanelMode />;
            case resourceTypes.CLUSTER:
                return <Cluster entityId={entityId} sidePanelMode />;
            case resourceTypes.DEPLOYMENT:
                return <Deployment entityId={entityId} sidePanelMode />;
            case standardEntityTypes.CONTROL:
                return <Control entityId={entityId} sidePanelMode />;
            default:
                return null;
        }
    }

    function closeSidePanel() {
        const baseURL = URLService.getURL(match, location).clearSidePanelParams().url();
        navigate(baseURL);
    }
    const headerUrl = URLService.getURL(match, location).base(entityType, entityId).url();

    return (
        <Query query={entityNameQueryMap[entityType]} variables={{ id: entityId }}>
            {({ loading, data }) => {
                let linkText = 'loading...';
                if (!loading && data) {
                    linkText = truncate(getEntityName(entityType, data), MAX_CONTROL_TITLE);
                }
                const headerTextComponent = (
                    <div className="w-full flex items-center">
                        <div className="flex items-center" data-testid="side-panel-header">
                            <Link to={headerUrl} className="w-full flex ml-4">
                                <div className="flex flex-1 items-center leading-normal">
                                    {linkText}
                                </div>
                            </Link>
                            <Link
                                className="mx-4 p-1 rounded flex"
                                to={headerUrl}
                                target="_blank"
                                rel="noopener noreferrer"
                                aria-label="External link"
                            >
                                <ExternalLinkAltIcon />
                            </Link>
                        </div>
                    </div>
                );
                const entityPage = getEntityPage();
                return (
                    <PanelNew testid="side-panel">
                        <PanelHead>
                            {headerTextComponent}
                            <PanelHeadEnd>
                                <CloseButton
                                    onClose={closeSidePanel}
                                    className="border-base-400 border-l"
                                />
                            </PanelHeadEnd>
                        </PanelHead>
                        <PanelBody>{entityPage}</PanelBody>
                    </PanelNew>
                );
            }}
        </Query>
    );
};

SidePanel.propTypes = {
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
};

export default SidePanel;
