import React from 'react';
import PropTypes from 'prop-types';

export default function Descriptor({ header, description }) {
    return (
        <div className="bg-base-100 px-3">
            <div className="pin-t py-3 font-700 text-base text-base-600 border-b border-base-300">
                {header}
            </div>
            <div className="pin-t py-3 font-600 text-lg text-base-600 leading-normal">
                {description}
            </div>
        </div>
    );
}

Descriptor.propTypes = {
    header: PropTypes.string.isRequired,
    description: PropTypes.string.isRequired
};
