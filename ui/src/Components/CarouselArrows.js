import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

export const CarouselNextArrow = ({ onClick }) => (
    <button
        type="button"
        className="bg-base-100 block carousel-next-arrow h-10 hover:bg-primary-100 hover:text-primary-600 rounded-full text-base-500 w-10"
        onClick={onClick}
    >
        <Icon.ChevronRight className="mt-1" />
    </button>
);
CarouselNextArrow.propTypes = {
    onClick: PropTypes.func.isRequired
};

export const CarouselPrevArrow = ({ onClick }) => (
    <button
        type="button"
        className="bg-base-100 block carousel-prev-arrow h-10 hover:bg-primary-100 hover:text-primary-600 rounded-full text-base-500 w-10"
        onClick={onClick}
    >
        <Icon.ChevronLeft className="mt-1" />
    </button>
);
CarouselPrevArrow.propTypes = {
    onClick: PropTypes.func.isRequired
};
