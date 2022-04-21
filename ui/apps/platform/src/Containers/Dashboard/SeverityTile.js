import React from 'react';
import { Link } from 'react-router-dom';
import PropTypes from 'prop-types';

import { severityLabels } from 'messages/common';
import { useTheme } from 'Containers/ThemeProvider';

const SeverityTile = ({ severity, count, index, color, link }) => {
    const { isDarkMode } = useTheme();
    function renderTileContent() {
        const backgroundStyle = {
            backgroundColor: color,
        };

        return (
            <div>
                <div className="absolute left-0 top-0 m-2">
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
                className={`${
                    !isDarkMode ? 'severity-tile border-base-100' : 'border-base-0'
                }  flex flex-1 flex-col border-3 p-4 text-center rounded-sm relative ${
                    index !== 0 ? 'ml-4' : ''
                }`}
                data-testid="severity-tile"
            >
                {renderTileContent()}
            </div>
        );
    }
    return (
        <Link
            className={`${
                !isDarkMode ? 'severity-tile border-base-100' : 'border-base-0'
            }  flex flex-1 flex-col border-3 p-4 text-center relative cursor-pointer rounded-sm no-underline hover:bg-primary-200 hover:shadow hover:bg-base-200 ${
                index !== 0 ? 'ml-4' : ''
            }`}
            to={link}
            data-testid="severity-tile"
        >
            {renderTileContent()}
        </Link>
    );
};

SeverityTile.propTypes = {
    severity: PropTypes.string.isRequired,
    count: PropTypes.number.isRequired,
    index: PropTypes.number.isRequired,
    color: PropTypes.string.isRequired,
    link: PropTypes.string.isRequired,
};

export default SeverityTile;
