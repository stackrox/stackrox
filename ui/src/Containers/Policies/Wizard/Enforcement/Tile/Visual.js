import React from 'react';
import PropTypes from 'prop-types';

export default function Visual(props) {
    return (
        <div className="flex flex-col h-full py-2">
            <div className="flex justify-center">
                <img src={props.image} alt={props.label} />
            </div>
            <div className="flex py-2 font-700 text-primary-700 justify-center">{props.label}</div>
        </div>
    );
}

Visual.propTypes = {
    image: PropTypes.string.isRequired,
    label: PropTypes.string.isRequired
};
