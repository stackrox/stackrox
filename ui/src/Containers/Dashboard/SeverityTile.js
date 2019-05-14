import React from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

import { severityLabels } from 'messages/common';

const SeverityTile = ({ severity, count, index, color }) => {
    function renderTileContent() {
        const backgroundStyle = {
            backgroundColor: color
        };
        return (
            <div>
                <div className="absolute pin-l pin-t m-2">
                    <div className="h-3 w-3 border-2 border-base-100" style={backgroundStyle} />
                </div>
                <div className="text-6xl font-sans text-primary-800 mb-2">{count}</div>
                <div className="uppercase tracking-wide text-base font-700 font-sans text-primary-800">
                    {severityLabels[severity]}
                </div>
            </div>
        );
    }

    if (count === 0) {
        return (
            <div
                className={`severity-tile flex flex-1 flex-col border-base-100 border-3 p-4 text-center rounded-sm relative ${
                    index !== 0 ? 'ml-4' : ''
                }`}
            >
                {renderTileContent()}
            </div>
        );
    }
    return (
        <Link
            className={`severity-tile flex flex-1 flex-col border-3 border-base-100 p-4 text-center relative cursor-pointer rounded-sm no-underline hover:bg-primary-200 hover:shadow hover:bg-base-200 ${
                index !== 0 ? 'ml-4' : ''
            }`}
            to={`/main/violations?severity=${severity}`}
        >
            {renderTileContent()}
        </Link>
    );
};

SeverityTile.propTypes = {
    severity: PropTypes.string.isRequired,
    count: PropTypes.number.isRequired,
    index: PropTypes.number.isRequired,
    color: PropTypes.string.isRequired
};

export default SeverityTile;
