import React from 'react';
import entityLabels from 'messages/entity';
import pluralize from 'pluralize';
import URLService from 'modules/URLService';
import { Link } from 'react-router-dom';
import { ChevronRight } from 'react-feather';
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
    entityId2,
    entityType2
}) => {
    const breadCrumbStates = [];
    if (entityType1 && entityId1) {
        breadCrumbStates.push({ name: entityName, type: entityLabels[entityType1] });
    }
    if (entityListType2) {
        breadCrumbStates.push({
            name: pluralize(entityLabels[entityListType2]),
            type: 'entity list'
        });
    }
    if (entityId2) {
        breadCrumbStates.push({
            name: relatedEntityName,
            type: entityLabels[entityType2] || entityLabels[entityListType2]
        });
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
    let maxWidthClass = 'max-w-full';
    if (breadCrumbStates.length > 1) maxWidthClass = `max-w-1/${breadCrumbStates.length}`;
    const breadCrumbLinks = breadCrumbStates.map((state, i, { length }) => {
        const icon = i !== length - 1 ? Icon : null;
        const link = getLink(match, location, i, length);
        const content = link ? (
            <Link
                className="text-primary-700 truncate uppercase truncate"
                title={state.name}
                to={link}
            >
                {state.name}
            </Link>
        ) : (
            <span className="w-full truncate" title={state.name}>
                <span className="truncate uppercase">{state.name}</span>
            </span>
        );
        if (!state) return null;
        return (
            <div key={`${state.name}--${state.type}`} className={`flex ${maxWidthClass} truncate`}>
                <span className="flex flex-col max-w-full" data-test-id="breadcrumb-link-text">
                    {content}
                    <span className="capitalize italic font-600">{state.type.toLowerCase()}</span>
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

export default BreadCrumbLinks;
