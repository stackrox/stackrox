import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

const Dot = ({ active, onClick }) => (
    <>
        <button
            type="button"
            onClick={onClick}
            className={`bg-base-300 h-2 w-2 ml-1 mr-1 rounded-full ${
                active ? 'bg-primary-400' : ''
            }`}
        />
    </>
);

export const PagerDots = ({ onPageChange, pageCount, currentPage }) => {
    const handleSetPage = page => () => {
        if (page < 0 || page >= pageCount) return;

        onPageChange(page);
    };
    return (
        <div className="absolute z-10 pin-r pin-b m-2">
            {Array(pageCount)
                .fill()
                .map((_, page) => (
                    <Dot
                        key={page.toString()}
                        active={page === currentPage}
                        onClick={handleSetPage(page)}
                    />
                ))}
        </div>
    );
};

PagerDots.propTypes = {
    onPageChange: PropTypes.func.isRequired,
    pageCount: PropTypes.number.isRequired,
    currentPage: PropTypes.number
};

PagerDots.defaultProps = {
    currentPage: 0
};

export const PagerButtonGroup = ({ onPagePrev, onPageNext }) => (
    <div>
        <button
            type="button"
            onClick={onPagePrev}
            className="border-base-300 border-l-2 border-t-2 border-b-2 rounded-sm hover:bg-base-200"
        >
            <Icon.ChevronLeft className="mt-1 h-4" />
        </button>
        <button
            type="button"
            onClick={onPageNext}
            className="border-base-300 border-2 hover:bg-base-200"
        >
            <Icon.ChevronRight className="mt-1 h-4" />
        </button>
    </div>
);

PagerButtonGroup.propTypes = {
    onPagePrev: PropTypes.func.isRequired,
    onPageNext: PropTypes.func.isRequired
};
