import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { ClipLoader as Loader } from 'react-spinners';
import TileContent from 'Components/TileContent';

const SummaryTileCount = ({ label, value, loading }) => {
    return (
        <li
            key={label}
            className="flex flex-col border-r border-base-400 border-dashed px-3 lg:w-24 md:w-20 no-underline py-3 text-base-500 items-center justify-center font-condensed"
            data-testid="summary-tile-count"
        >
            {loading && !value ? (
                <Loader loading size={12} color="currentColor" />
            ) : (
                <TileContent
                    superText={value}
                    text={pluralize(label, value)}
                    textColorClass="text-base-500"
                />
            )}
        </li>
    );
};

SummaryTileCount.propTypes = {
    label: PropTypes.string.isRequired,
    value: PropTypes.number,
    loading: PropTypes.bool
};

SummaryTileCount.defaultProps = {
    loading: false,
    value: 0
};

export default SummaryTileCount;
