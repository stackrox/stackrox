import React, { ReactElement, useContext } from 'react';
import { Link } from 'react-router-dom';
import { ChevronRight, ArrowLeft } from 'react-feather';

import entityTypes from 'constants/entityTypes';
import EntityIcon from 'Components/EntityIcon';
import workflowStateContext from 'Containers/workflowStateContext';
import { useTheme } from 'Containers/ThemeProvider';

import EntityBreadCrumb, { WorkflowEntity } from './EntityBreadCrumb';

const Icon = (
    <ChevronRight className="bg-base-200 border border-base-400 mx-4 rounded-full" size="14" />
);

const BackLink = ({ workflowState, enabled }) => {
    const { isDarkMode } = useTheme();

    // if the link for this particular crumb is enabled, calculate the URL
    //   necessary to go back up the stack,
    //   and remove any existing sort and search on a sublist (fixes https://stack-rox.atlassian.net/browse/ROX-4449)
    const url = !enabled ? null : workflowState.pop().clearSort().clearSearch().toUrl();

    return url ? (
        <Link
            className="flex items-center justify-center text-base-600 border-r border-base-300 px-4 mr-4 h-full hover:bg-primary-200 w-16"
            to={url}
            aria-label="Go to preceding breadcrumb"
        >
            <ArrowLeft className="h-6 w-6 text-600" />
        </Link>
    ) : (
        <EntityIcon
            className={`flex items-center justify-center border-r  px-4 mr-4 h-full w-16 ${
                !isDarkMode ? 'border-base-300' : 'border-base-400'
            }`}
            entityType={workflowState.getCurrentEntity().entityType}
        />
    );
};

const getUrl = (workflowState, steps) => {
    // TODO: do this with .call
    let newState = workflowState;
    for (let x = 1; x < steps; x += 1) {
        newState = newState.pop().clearSort().clearSearch();
    }
    const newURL = newState.toUrl() as string;
    const currentURL = workflowState.toUrl() as string;
    return newURL === currentURL ? null : newURL;
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

export type EntityBreadCrumbsProps = {
    workflowEntities: WorkflowEntity[];
};

function EntityBreadCrumbs({ workflowEntities }: EntityBreadCrumbsProps): ReactElement {
    const workflowState = useContext(workflowStateContext);

    const maxWidthClass = getMaxWidthClass(workflowEntities.length);

    const breadCrumbLinks = workflowEntities.map((workflowEntity, i, { length }) => {
        const icon = i !== length - 1 ? Icon : null;
        const url = getUrl(workflowState, length - i);
        const { entityType, entityId } = workflowEntity;

        const extraClasses = entityType === entityTypes.IMAGE ? '' : `${maxWidthClass} truncate`;

        return (
            <div key={`${entityType}-${entityId}`} className={`flex ${extraClasses}`}>
                <EntityBreadCrumb workflowEntity={workflowEntity} url={url} />
                <span className="flex items-center">{icon}</span>
            </div>
        );
    });
    const backButtonEnabled = !!(workflowEntities.length > 1);
    return (
        <span
            style={{ flex: '10 1' }}
            className="flex items-center leading-normal text-base-600 truncate"
        >
            <BackLink workflowState={workflowState} enabled={backButtonEnabled} />
            {breadCrumbLinks}
        </span>
    );
}

export default EntityBreadCrumbs;
