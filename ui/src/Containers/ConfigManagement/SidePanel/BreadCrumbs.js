import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import URLService from 'modules/URLService';
import { Link, withRouter } from 'react-router-dom';
import getEntityName from 'modules/getEntityName';
import { entityNameQueryMap } from 'modules/queryMap';

import { ChevronRight } from 'react-feather';
import Query from 'Components/ThrowingQuery';
import BackButton from 'Containers/ConfigManagement/SidePanel/buttons/BackButton';

const Icon = (
    <ChevronRight className="bg-base-200 border border-base-400 mx-4 rounded-full" size="14" />
);

const getBreadCrumbStates = ({
    entityName,
    relatedEntityName,
    entityType1,
    entityId1,
    entityListType2,
    entityId2
}) => {
    const breadCrumbStates = [];
    if (entityType1 && entityId1) {
        breadCrumbStates.push(entityName);
    }
    if (entityListType2) breadCrumbStates.push(pluralize(entityLabels[entityListType2]));
    if (entityId2) {
        breadCrumbStates.push(relatedEntityName);
    }
    return breadCrumbStates;
};

const getLink = (match, location, index, length) => {
    const numPops = length - 1 - index;
    if (!numPops || numPops < 0) return null;
    const urlBuilder = URLService.getURL(match, location);
    for (let j = 0; j < numPops; j += 1) {
        urlBuilder.pop();
    }
    return urlBuilder.url();
};

const BreadCrumbLinks = props => {
    const { className, match, location, history, ...params } = props;
    const { entityType1, entityId1, entityListType2, entityId2 } = params;
    if (!entityId1) return null;
    const breadCrumbStates = getBreadCrumbStates(params);
    const breadCrumbLinks = breadCrumbStates.map((state, i, { length }) => {
        const icon = i !== length - 1 ? Icon : null;
        const link = getLink(match, location, i, length);
        const content = link ? (
            <Link className="text-primary-700" to={link}>
                {state}
            </Link>
        ) : (
            state
        );
        if (!state) return null;
        return (
            <span className="flex items-center" key={i} data-test-id="breadcrumb-link-text">
                {content} {icon}
            </span>
        );
    });
    return (
        <span className={`flex items-center ${className}`}>
            <BackButton
                entityType1={entityType1}
                entityListType2={entityListType2}
                entityId2={entityId2}
            />
            {breadCrumbLinks}
        </span>
    );
};

BreadCrumbLinks.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    history: ReactRouterPropTypes.history.isRequired,
    className: PropTypes.string
};

BreadCrumbLinks.defaultProps = {
    className: ''
};

const BreadCrumbs = props => {
    const { className, match, location, ...params } = props;
    const { entityType1, entityId1, entityType2, entityListType2, entityId2 } = params;
    if (!entityId1) return null;

    const entityQuery = entityNameQueryMap[entityType1];
    const entityVariables = { id: entityId1 };

    const relatedEntityType = entityListType2 || entityType2;
    const relatedEntityQuery = entityNameQueryMap[relatedEntityType];
    const relatedEntityVariables = { id: entityId2 };

    return (
        <Query query={entityQuery} variables={entityVariables}>
            {({ loading: entityLoading, data: entityData }) => {
                if (!entityLoading && !entityData) return null;
                const entityName = getEntityName(entityType1, entityData);
                if (!entityId2) {
                    return <BreadCrumbLinks {...props} entityName={entityName} />;
                }
                return (
                    <Query query={relatedEntityQuery} variables={relatedEntityVariables}>
                        {({ loading: relatedEntityLoading, data: relatedEntityData }) => {
                            if (!relatedEntityLoading && !relatedEntityData) return null;
                            const relatedEntityName = getEntityName(
                                relatedEntityType,
                                relatedEntityData
                            );
                            return (
                                <BreadCrumbLinks
                                    {...props}
                                    entityName={entityName}
                                    relatedEntityName={relatedEntityName}
                                />
                            );
                        }}
                    </Query>
                );
            }}
        </Query>
    );
};

BreadCrumbs.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    className: PropTypes.string,
    entityType1: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string
};

BreadCrumbs.defaultProps = {
    className: '',
    entityType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null
};

export default withRouter(BreadCrumbs);
