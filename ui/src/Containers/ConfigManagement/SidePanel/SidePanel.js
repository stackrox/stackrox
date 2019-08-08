import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';

import { ExternalLink as ExternalLinkIcon } from 'react-feather';
import Button from 'Components/Button';
import Panel from 'Components/Panel';
import searchContext from 'Containers/searchContext';
import EntityPage from 'Containers/ConfigManagement/Entity';
import ReactRouterPropTypes from 'react-router-prop-types';
import BreadCrumbs from './BreadCrumbs';

const ExternalLink = ({ onClick }) => (
    <div className="flex items-center h-full hover:bg-base-300">
        <Button
            dataTestId="external-link"
            className="border-l border-base-300 h-full px-4"
            icon={<ExternalLinkIcon className="h-6 w-6 text-base-600" />}
            onClick={onClick}
        />
    </div>
);

ExternalLink.propTypes = {
    onClick: PropTypes.func.isRequired
};

const SidePanel = ({
    match,
    location,
    history,
    className,
    contextEntityType,
    contextEntityId,
    entityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query
}) => {
    const searchParam = useContext(searchContext);
    const isList = !entityId1 || ((entityType2 || entityListType2) && !entityId2);

    function getCurrentEntityId() {
        return entityId2 || entityId1;
    }

    function getCurrentEntityType() {
        if (isList) return entityType1;
        return entityType2 || entityListType2 || entityType1;
    }

    function getListType() {
        if (!isList) return null;
        return entityListType2;
    }

    function getSearchParams() {
        return query[searchParam];
    }

    function onExternalLinkClick() {
        const url = URLService.getURL(match, location)
            .base(getCurrentEntityType(), getCurrentEntityId())
            .query()
            .query(getSearchParams())
            .url();
        history.push(url);
    }

    function onClose() {
        history.push(
            URLService.getURL(match, location)
                .clearSidePanelParams()
                .url()
        );
    }

    const entityId = getCurrentEntityId();
    const entityType = getCurrentEntityType();
    const listType = getListType();
    const entityContext = {};
    if (contextEntityType) entityContext[contextEntityType] = contextEntityId;
    if (entityId2) entityContext[entityType1] = entityId1;

    return (
        <div className={className}>
            <Panel
                id="side-panel"
                bodyClassName={`${entityListType2 ? 'bg-base-100' : 'bg-primary-100'}`}
                headerTextComponent={
                    <BreadCrumbs
                        className="font-700 leading-normal text-base-600 uppercase tracking-wide"
                        entityType1={entityType1}
                        entityId1={entityId1}
                        entityType2={entityType2}
                        entityListType2={entityListType2}
                        entityId2={entityId2}
                    />
                }
                headerComponents={<ExternalLink onClick={onExternalLinkClick} />}
                onClose={onClose}
            >
                <EntityPage
                    entityContext={entityContext}
                    entityType={entityType}
                    entityId={entityId}
                    entityListType={listType}
                    query={query}
                />
            </Panel>
        </div>
    );
};

SidePanel.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    className: PropTypes.string,
    contextEntityType: PropTypes.string,
    contextEntityId: PropTypes.string,
    entityType1: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
    query: PropTypes.shape().isRequired
};

SidePanel.defaultProps = {
    className: '',
    contextEntityType: null,
    contextEntityId: null,
    entityType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null
};

export default withRouter(SidePanel);
