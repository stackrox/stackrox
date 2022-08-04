import React from 'react';
import { Link } from 'react-router-dom';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';

import dateFns from 'date-fns';
import NoResultsMessage from 'Components/NoResultsMessage';

const renderMoreButton = (deployments) => {
    if (!deployments.length) {
        return null;
    }
    return (
        <Link to="/main/risk" className="no-underline">
            <button
                type="button"
                className="border-2 border-base-400 font-700 hover:bg-base-200 hover:border-primary-400 hover:text-primary-700 p-1 rounded-sm text-base-600 text-xs uppercase"
            >
                View All
            </button>
        </Link>
    );
};

const renderDeploymentsList = (deployments) => {
    if (!deployments.length) {
        return (
            <NoResultsMessage message="No data available. Please ensure your cluster is properly configured." />
        );
    }
    const list = deployments.map((deployment) => (
        <li key={deployment.id}>
            <Link
                to={`/main/risk/${deployment.id}`}
                className="no-underline flex justify-between border-b p-4 border-base-300 hover:bg-base-200"
            >
                <div className="text-base-600">{deployment.name}</div>
                <div className="text-base-500 font-600">
                    <span className="pr-1 border-r inline-block">
                        {dateFns.format(deployment.created, 'MM/DD')}
                    </span>
                    <span className="pl-1">{dateFns.format(deployment.created, 'h:mm:ss A')}</span>
                </div>
            </Link>
        </li>
    ));
    return <ul className="h-full">{list}</ul>;
};

const TopRiskyDeployments = ({ deployments }) => {
    if (!deployments) {
        return '';
    }
    return (
        <div
            className="flex flex-col bg-base-100 rounded shadow h-full"
            data-testid="top-risky-deployments"
        >
            <h2 className="flex items-center text-lg text-base font-sans text-base-600 tracking-wide border-base-300 border-b">
                <Icon.File className="h-4 w-4 m-3" />
                <span className="px-4 py-4 pl-3 uppercase text-base tracking-wide pb-3 border-l border-base-300">
                    Top Risky Deployments
                </span>
                <span className="flex flex-1 justify-end pr-2">
                    {renderMoreButton(deployments)}
                </span>
            </h2>
            <div className="m-4 h-64">{renderDeploymentsList(deployments)}</div>
        </div>
    );
};

TopRiskyDeployments.propTypes = {
    deployments: PropTypes.arrayOf(
        PropTypes.shape({
            deployment: PropTypes.shape({}),
        })
    ).isRequired,
};

export default TopRiskyDeployments;
