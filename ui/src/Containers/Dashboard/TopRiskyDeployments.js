import React from 'react';
import { Link } from 'react-router-dom';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import dateFns from 'date-fns';

const TopRiskyDeployments = props => (
    <div className="flex flex-col p-4 bg-white rounded-sm shadow">
        <h2 className="flex items-center text-lg text-base font-sans text-base-600 pb-4 tracking-wide border-primary-200 border-b">
            <Icon.File className="h-4 w-4 mr-3" />
            Top Risky Deployments
            <span className="flex-1 self-end text-right">
                <Link to="/main/risk" className="no-underline">
                    <button className="border px-2 py-2 text-base-400 border-primary-200 text-sm hover:bg-base-400 hover:text-white hover:border-base-400">
                        More
                    </button>
                </Link>
            </span>
        </h2>
        <ul className="p-0 list-reset">
            {props.deployments.map(deployment => (
                <li key={deployment.id}>
                    <Link
                        to={`/main/risk/${deployment.id}`}
                        className="no-underline flex flex-row justify-between border-b p-4 border-primary-200 hover:bg-base-100"
                    >
                        <div className="text-base-600">{deployment.name}</div>
                        <div className="text-base-400 font-400">
                            <span className="pr-1 border-r inline-block">
                                {dateFns.format(deployment.updatedAt, 'MM/DD')}
                            </span>
                            <span className="pl-1">
                                {dateFns.format(deployment.updatedAt, 'h:mm:ss A')}
                            </span>
                        </div>
                    </Link>
                </li>
            ))}
        </ul>
    </div>
);

TopRiskyDeployments.propTypes = {
    deployments: PropTypes.arrayOf(PropTypes.object).isRequired
};

export default TopRiskyDeployments;
