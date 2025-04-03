import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { useLocation, useNavigate, Link } from 'react-router-dom';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';

import CloseButton from 'Components/CloseButton';
import { PanelNew, PanelBody, PanelHead, PanelHeadEnd } from 'Components/Panel';
import searchContext from 'Containers/searchContext';
import Entity from 'Containers/ConfigManagement/Entity';
import workflowStateContext from 'Containers/workflowStateContext';
import useWorkflowMatch from 'hooks/useWorkflowMatch';
import parseURL from 'utils/URLParser';
import URLService from 'utils/URLService';
import BreadCrumbs from './BreadCrumbs';

const SidePanel = ({
    contextEntityType,
    contextEntityId,
    entityListType1,
    entityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query,
}) => {
    const match = useWorkflowMatch();
    const location = useLocation();
    const navigate = useNavigate();
    const workflowState = parseURL(location);
    const searchParam = useContext(searchContext);
    const isList = !entityId1 || (entityListType2 && !entityId2);

    function getCurrentEntityId() {
        return entityId2 || entityId1;
    }

    function getCurrentEntityType() {
        return (
            entityType2 ||
            (entityId2 && entityListType2) ||
            entityType1 ||
            entityListType1 ||
            contextEntityType
        );
    }

    function getListType() {
        if (!isList) {
            return null;
        }
        return entityListType2;
    }

    function getSearchParams() {
        return query[searchParam];
    }

    function onClose() {
        navigate(URLService.getURL(match, location).clearSidePanelParams().url());
    }

    const entityId = getCurrentEntityId();
    const entityType = getCurrentEntityType();
    const listType = getListType();
    const externalURL = URLService.getURL(match, location)
        .base(entityType, entityId)
        .push(listType)
        .query()
        .query(getSearchParams())
        .url();
    const externalLink = (
        <div className="flex items-center h-full">
            <Link
                to={externalURL}
                aria-label="link"
                className="border-base-400 border-l h-full p-4"
            >
                <ExternalLinkAltIcon />
            </Link>
        </div>
    );

    const entityContext = {};
    if (contextEntityType) {
        entityContext[contextEntityType] = contextEntityId;
    }
    if (entityId2) {
        entityContext[entityType1 || entityListType1] = entityId1;
    }
    return (
        <workflowStateContext.Provider value={workflowState}>
            <PanelNew testid="side-panel">
                <PanelHead>
                    <BreadCrumbs
                        className="leading-normal text-base-600 truncate"
                        entityType1={entityType1 || entityListType1}
                        entityId1={entityId1}
                        entityType2={entityType2}
                        entityListType2={entityListType2}
                        entityId2={entityId2}
                    />
                    <PanelHeadEnd>
                        {externalLink}
                        <CloseButton onClose={onClose} className="border-base-400 border-l" />
                    </PanelHeadEnd>
                </PanelHead>
                <PanelBody>
                    <Entity
                        entityContext={entityContext}
                        entityType={entityType}
                        entityId={entityId}
                        entityListType={listType}
                        query={query}
                    />
                </PanelBody>
            </PanelNew>
        </workflowStateContext.Provider>
    );
};

SidePanel.propTypes = {
    contextEntityType: PropTypes.string,
    contextEntityId: PropTypes.string,
    entityType1: PropTypes.string,
    entityListType1: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
    query: PropTypes.shape().isRequired,
};

SidePanel.defaultProps = {
    contextEntityType: null,
    contextEntityId: null,
    entityType1: null,
    entityListType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null,
};

export default SidePanel;
