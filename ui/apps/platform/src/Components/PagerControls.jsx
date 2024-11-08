import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

export const PagerButtonGroup = ({ onPagePrev, onPageNext, enableNext, enablePrev }) => (
    <div className="-mt-1 flex">
        <button
            type="button"
            onClick={onPagePrev}
            disabled={!enablePrev}
            className={`border-base-300 border-l-2 border-t-2 border-b-2 rounded-sm hover:bg-base-200 ${
                !enableNext ? 'border-r-2' : ''
            }`}
        >
            <Icon.ChevronLeft className="mt-1 h-4" />
        </button>
        <button
            type="button"
            onClick={onPageNext}
            disabled={!enableNext}
            className="border-base-300 border-2 hover:bg-base-200"
        >
            <Icon.ChevronRight className="mt-1 h-4" />
        </button>
    </div>
);

PagerButtonGroup.propTypes = {
    onPagePrev: PropTypes.func.isRequired,
    onPageNext: PropTypes.func.isRequired,
    enableNext: PropTypes.bool.isRequired,
    enablePrev: PropTypes.bool.isRequired,
};
