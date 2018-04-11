import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

export function CarouselNextArrow(props) {
    const { onClick } = props;
    return (
        <button
            className="border border-base-300 text-base-500 carousel-next-arrow px-2 py-2 hover:text-white hover:bg-base-300 bg-white block"
            onClick={onClick}
        >
            <Icon.ChevronRight className="w-4 h-4" />
        </button>
    );
}

CarouselNextArrow.defaultProps = {
    onClick: PropTypes.func
};

CarouselNextArrow.propTypes = {
    onClick: PropTypes.func
};

export function CarouselPrevArrow(props) {
    const { onClick } = props;
    return (
        <button
            className="border border-base-300 text-base-500 carousel-prev-arrow px-2 py-2 hover:text-white hover:bg-base-300 bg-white block"
            onClick={onClick}
        >
            <Icon.ChevronLeft className="w-4 h-4" />
        </button>
    );
}

CarouselPrevArrow.defaultProps = {
    onClick: PropTypes.func
};

CarouselPrevArrow.propTypes = {
    onClick: PropTypes.func
};
