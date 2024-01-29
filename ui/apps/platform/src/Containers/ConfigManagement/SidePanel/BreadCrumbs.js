import React from 'react';
import ReactRouterPropTypes from 'react-router-prop-types';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import upperFirst from 'lodash/upperFirst';
import { ChevronRight } from 'react-feather';
import { Link, withRouter } from 'react-router-dom';

import useEntityName from 'hooks/useEntityName';
import entityLabels from 'messages/entity';
import URLService from 'utils/URLService';

import BackButton from './BackButton';

const Icon = (
    <ChevronRight className="bg-base-200 border border-base-400 mx-4 rounded-full" size="14" />
);

const getBreadCrumbStates = ({
    entityName,
    relatedEntityName,
    entityType1,
    entityId1,
    entityListType2,
    entityId2,
    entityType2,
}) => {
    const breadCrumbStates = [];
    if (entityType1 && entityId1) {
        breadCrumbStates.push({ name: entityName, type: entityLabels[entityType1] });
    }
    if (entityListType2) {
        breadCrumbStates.push({
            name: pluralize(entityLabels[entityListType2]),
            type: 'entity list',
        });
    }
    if (entityId2) {
        breadCrumbStates.push({
            name: relatedEntityName,
            type: entityLabels[entityType2] || entityLabels[entityListType2],
        });
    }
    return breadCrumbStates;
};

const getLink = (match, location, index, length) => {
    const numPops = length - 1 - index;
    if (!numPops || numPops < 0) {
        return null;
    }
    const urlBuilder = URLService.getURL(match, location);
    for (let j = 0; j < numPops; j += 1) {
        urlBuilder.pop();
    }
    return urlBuilder.url();
};

// Tailwind purge needs to see complete class strings instead of `max-w-1/${length}` template literal.
const getMaxWidthClass = (length) => {
    switch (length) {
        case 1:
            return 'max-w-full';
        case 2:
            return 'max-w-1/2';
        case 3:
            return 'max-w-1/3';
        case 4:
            return 'max-w-1/4';
        case 5:
            return 'max-w-1/5';
        case 6:
            return 'max-w-1/6';
        case 7:
            return 'max-w-1/7';
        case 8:
            return 'max-w-1/8';
        case 9:
            return 'max-w-1/9';
        case 10:
            return 'max-w-1/10';
        default:
            return '';
    }
};

const BreadCrumbLinks = (props) => {
    // disable because unused history might be specified for rest spread idiom.
    /* eslint-disable no-unused-vars */
    const { className, match, location, history, ...params } = props;
    /* eslint-enable no-unused-vars */
    const { entityType1, entityId1, entityListType2, entityId2 } = params;
    if (!entityId1) {
        return null;
    }
    const breadCrumbStates = getBreadCrumbStates(params);
    const maxWidthClass = getMaxWidthClass(breadCrumbStates.length);
    const breadCrumbLinks = breadCrumbStates.map((state, i, { length }) => {
        const icon = i !== length - 1 ? Icon : null;
        const link = getLink(match, location, i, length);
        const name = state.type === 'entity list' ? upperFirst(state.name) : state.name;
        const content = link ? (
            <Link
                className="text-primary-700 underline truncate font-700"
                title={state.name}
                to={link}
            >
                {name}
            </Link>
        ) : (
            <span className="w-full truncate" title={state.name}>
                <span className="truncate font-700">{name}</span>
            </span>
        );
        if (!state) {
            return null;
        }
        const entityTypeLabel = upperFirst(state.type);
        return (
            <div key={`${state.name}--${state.type}`} className={`flex ${maxWidthClass} truncate`}>
                <span className="flex flex-col max-w-full" data-testid="breadcrumb-link-text">
                    {content}
                    <span>{entityTypeLabel}</span>
                </span>
                <span className="flex items-center">{icon}</span>
            </div>
        );
    });
    return (
        <span style={{ flex: '10 1' }} className={`flex items-center ${className}`}>
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
    className: PropTypes.string,
};

BreadCrumbLinks.defaultProps = {
    className: '',
};

const BreadCrumbs = (props) => {
    // disable because unused className, match, location might be specified for rest spread idiom.
    /* eslint-disable no-unused-vars */
    const { className, match, location, ...params } = props;
    /* eslint-enable no-unused-vars */
    const { entityType1, entityId1, entityType2, entityListType2, entityId2 } = params;

    const relatedEntityType = entityListType2 || entityType2;

    const { loading: entityLoading, entityName: mainEntityName } = useEntityName(
        entityType1,
        entityId1
    );
    const { loading: relatedEntityLoading, entityName: childEntityName } = useEntityName(
        relatedEntityType,
        entityId2
    );

    if (!entityLoading && !mainEntityName) {
        return null;
    }
    if (!entityId2) {
        return <BreadCrumbLinks {...props} entityName={mainEntityName} />;
    }

    if (!relatedEntityLoading && !childEntityName) {
        return null;
    }
    return (
        <BreadCrumbLinks
            {...props}
            entityName={mainEntityName}
            relatedEntityName={childEntityName}
        />
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
    entityId2: PropTypes.string,
};

BreadCrumbs.defaultProps = {
    className: '',
    entityType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null,
};

export default withRouter(BreadCrumbs);
