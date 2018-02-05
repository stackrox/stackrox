import React from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

import { severityLabels } from 'messages/common';

const SeverityTile = (props) => {
    const backgroundStyle = {
        backgroundColor: props.color
    };
    return (
        <Link
            className={`flex flex-1 flex-col bg-white border border-base-300 p-4 text-center relative cursor-pointer no-underline hover:border-base-500 hover:shadow hover:bg-base-100 ${props.index !== 0 ? 'ml-4' : ''}`}
            to={`/violations?severity=${props.severity}`}
        >
            <div className="absolute pin-l pin-t m-2">
                <div className="h-3 w-3" style={backgroundStyle} />
            </div>
            <div className="text-4xl text-base font-sans text-primary-500">{props.count}</div>
            <div className="text-lg text-base font-sans text-primary-500">{severityLabels[props.severity]}</div>
        </Link>
    );
};

SeverityTile.propTypes = {
    severity: PropTypes.string.isRequired,
    count: PropTypes.number.isRequired,
    color: PropTypes.string.isRequired,
    index: PropTypes.number.isRequired
};

export default SeverityTile;
