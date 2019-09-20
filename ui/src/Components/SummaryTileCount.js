import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { ClipLoader as Loader } from 'react-spinners';

const SummaryTileCount = ({ label, value, loading }) => {
    return (
        <li
            key={label}
            className="flex flex-col border-r border-base-400 border-dashed px-3 lg:w-24 md:w-20 no-underline py-3 text-base-500 items-center justify-center font-condensed"
        >
            <div className="text-3xl tracking-widest">
                {loading && !value ? <Loader loading size={12} color="currentColor" /> : value}
            </div>
            <div className="text-sm pt-1 tracking-wide">{pluralize(label, value)}</div>
        </li>
    );
};

SummaryTileCount.propTypes = {
    label: PropTypes.string.isRequired,
    value: PropTypes.number.isRequired,
    loading: PropTypes.bool
};

SummaryTileCount.defaultProps = {
    loading: false
};

export default SummaryTileCount;
