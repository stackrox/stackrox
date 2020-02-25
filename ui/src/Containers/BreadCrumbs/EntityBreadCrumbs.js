import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import { Link } from 'react-router-dom';
import { ChevronRight, ArrowLeft } from 'react-feather';
import EntityBreadCrumb from 'Containers/BreadCrumbs/EntityBreadCrumb';
import EntityIcon from 'Components/EntityIcon';
import workflowStateContext from 'Containers/workflowStateContext';

const Icon = (
    <ChevronRight className="bg-base-200 border border-base-400 mx-4 rounded-full" size="14" />
);

const BackLink = ({ workflowState, enabled }) => {
    const url = !enabled ? null : workflowState.pop().toUrl();
    return url ? (
        <Link
            className="flex items-center justify-center text-base-600 border-r border-base-300 px-4 mr-4 h-full hover:bg-primary-200 w-16"
            to={url}
            data-testid="sidepanelBackButton"
        >
            <ArrowLeft className="h-6 w-6 text-600" />
        </Link>
    ) : (
        <EntityIcon
            className="flex items-center justify-center border-r border-base-300 px-4 mr-4 h-full w-16"
            entityType={workflowState.getCurrentEntity().entityType}
        />
    );
};

const getUrl = (workflowState, steps) => {
    // TODO: do this with .call
    let newState = workflowState;
    for (let x = 1; x < steps; x += 1) {
        newState = newState.pop();
    }
    const newURL = newState.toUrl();
    const currentURL = workflowState.toUrl();
    return newURL === currentURL ? null : newURL;
};

const BreadCrumbLinks = ({ workflowEntities }) => {
    const workflowState = useContext(workflowStateContext);

    let maxWidthClass = 'max-w-full';

    if (workflowEntities.length > 1) maxWidthClass = `max-w-1/${workflowEntities.length}`;

    const breadCrumbLinks = workflowEntities.map((workflowEntity, i, { length }) => {
        const icon = i !== length - 1 ? Icon : null;
        const url = getUrl(workflowState, length - i);
        const { entityType, entityId } = workflowEntity;

        return (
            <div key={`${entityType}-${entityId}`} className={`flex ${maxWidthClass} truncate`}>
                <EntityBreadCrumb workflowEntity={workflowEntity} url={url} />
                <span className="flex items-center">{icon}</span>
            </div>
        );
    });
    const backButtonEnabled = !!(workflowEntities.length > 1);
    return (
        <span
            style={{ flex: '10 1' }}
            className="flex items-center font-700 leading-normal text-base-600 tracking-wide truncate"
        >
            <BackLink workflowState={workflowState} enabled={backButtonEnabled} />
            {breadCrumbLinks}
        </span>
    );
};

BreadCrumbLinks.propTypes = {
    workflowEntities: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

export default BreadCrumbLinks;
