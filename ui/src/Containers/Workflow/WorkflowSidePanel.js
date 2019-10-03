import React from 'react';
import { withRouter, Link } from 'react-router-dom';
import URLService from 'modules/URLService';
import onClickOutside from 'react-onclickoutside';
import { useTheme } from 'Containers/ThemeProvider';
import { useQuery } from 'react-apollo';

import { ExternalLink as ExternalLinkIcon } from 'react-feather';
import Panel from 'Components/Panel';

const WorkflowSidePanel = ({
    match,
    location,
    history,
    contextEntityType,
    entityListType1,
    entityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query,
    render
}) => {
    const { isDarkMode } = useTheme();
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
        if (!isList) return null;
        return entityListType2;
    }

    function onClose() {
        history.push(
            URLService.getURL(match, location)
                .clearSidePanelParams()
                .url()
        );
    }

    WorkflowSidePanel.handleClickOutside = () => {
        onClose();
    };

    const { loading, error, data } = useQuery(query, { variables: { id: entityId1 } });

    const entityId = getCurrentEntityId();
    const entityType = getCurrentEntityType();
    const listType = getListType();
    const externalURL = URLService.getURL(match, location)
        .base(entityType, entityId)
        .push(listType)
        .query()
        .url();
    const externalLink = (
        <div className="flex items-center h-full hover:bg-base-300">
            <Link
                to={externalURL}
                data-test-id="external-link"
                className={`${
                    !isDarkMode ? 'border-base-100' : 'border-base-400'
                } border-l h-full p-4`}
            >
                <ExternalLinkIcon className="h-6 w-6 text-base-600" />
            </Link>
        </div>
    );

    return (
        <div className="w-full h-full bg-base-100 border-l border-base-400 shadow-sidepanel">
            <Panel
                id="side-panel"
                headerClassName={`flex w-full h-14 overflow-y-hidden border-b ${
                    !isDarkMode ? 'bg-side-panel-wave border-base-100' : 'border-base-400'
                }`}
                bodyClassName={`${isList || isDarkMode ? 'bg-base-100' : ''}`}
                headerComponents={externalLink}
                onClose={onClose}
                closeButtonClassName={
                    isDarkMode ? 'border-l border-base-400' : 'border-l border-base-100'
                }
            >
                {render({ loading, error, data })}
            </Panel>
        </div>
    );
};

const clickOutsideConfig = {
    handleClickOutside: () => WorkflowSidePanel.handleClickOutside
};

/*
 * If more than one SidePanel is rendered, this Pure Functional Component will need to be converted to
 * a Class Component in order to work correctly. See https://github.com/stackrox/rox/pull/3090#pullrequestreview-274948849
 */
export default onClickOutside(withRouter(WorkflowSidePanel), clickOutsideConfig);
