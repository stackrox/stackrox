import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

export const CarouselNextArrow = ({ onClick }) => (
    <button
        className="border border-base-300 text-base-500 carousel-next-arrow px-2 py-2 hover:text-white hover:bg-base-300 bg-white block"
        onClick={onClick}
    >
        <Icon.ChevronRight className="w-4 h-4" />
    </button>
);
CarouselNextArrow.propTypes = {
    onClick: PropTypes.func.isRequired
};

export const CarouselPrevArrow = ({ onClick }) => (
    <button
        className="border border-base-300 text-base-500 carousel-prev-arrow px-2 py-2 hover:text-white hover:bg-base-300 bg-white block"
        onClick={onClick}
    >
        <Icon.ChevronLeft className="w-4 h-4" />
    </button>
);
CarouselPrevArrow.propTypes = {
    onClick: PropTypes.func.isRequired
};
