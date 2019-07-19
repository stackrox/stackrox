import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import { entityQueryKeys } from 'constants/entityTypes';

import { ExternalLink as ExternalLinkIcon } from 'react-feather';
import Button from 'Components/Button';
import Panel from 'Components/Panel';
import searchContext from 'Containers/searchContext';
import EntityOverview from 'Containers/ConfigManagement/Entity';
import List from 'Containers/ConfigManagement/EntityList';
import ReactRouterPropTypes from 'react-router-prop-types';
import BreadCrumbs from './BreadCrumbs';

const ExternalLink = ({ onClick }) => (
    <div className="flex items-center h-full hover:bg-base-300">
        <Button
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
    onClose,
    entityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query
}) => {
    const searchParam = useContext(searchContext);
    const isList = !entityId1 || ((entityType2 || entityListType2) && !entityId2);

    function onRelatedEntityClick(entityType, entityId) {
        const urlBuilder = URLService.getURL(match, location).push(entityType, entityId);
        history.push(urlBuilder.url());
    }

    function onRelatedEntityListClick(entityListType) {
        const urlBuilder = URLService.getURL(match, location).push(entityListType);
        history.push(urlBuilder.url());
    }

    function onRowClick(entityId) {
        const urlBuilder = URLService.getURL(match, location).push(entityId);
        history.push(urlBuilder.url());
    }

    function getCurrentEntityId() {
        if (isList) return null;
        return entityId2 || entityId1;
    }

    function getCurrentEntityType() {
        return entityType2 || entityListType2 || entityType1;
    }

    function getSearchParams() {
        return query[searchParam];
    }

    function getComponent() {
        const entityId = getCurrentEntityId();
        const entityType = getCurrentEntityType();

        if (!isList) {
            return (
                <EntityOverview
                    entityType={entityType}
                    entityId={entityId}
                    onRelatedEntityClick={onRelatedEntityClick}
                    onRelatedEntityListClick={onRelatedEntityListClick}
                />
            );
        }

        // Add entityId query parameter if there is another entity in the stack
        const panelQuery = {};
        if (entityId1) panelQuery[`${entityQueryKeys[entityType1]}`] = entityId1;

        return (
            <List
                entityListType={entityType}
                onRowClick={onRowClick}
                query={{ ...panelQuery, ...getSearchParams() }}
            />
        );
    }

    function onExternalLinkClick() {
        const url = URLService.getURL(match, location)
            .base(getCurrentEntityType(), getCurrentEntityId())
            .query()
            .query(getSearchParams())
            .url();
        history.push(url);
    }

    const component = getComponent();
    return (
        <div className={className}>
            <Panel
                bodyClassName="bg-primary-100"
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
                {component}
            </Panel>
        </div>
    );
};

SidePanel.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    className: PropTypes.string,
    entityType1: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
    onClose: PropTypes.func,
    query: PropTypes.shape().isRequired
};

SidePanel.defaultProps = {
    className: '',
    entityType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null,
    onClose: null
};

export default withRouter(SidePanel);
