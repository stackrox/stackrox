import React from 'react';
import PropTypes from 'prop-types';
import createInitials from 'initials';

function Avatar({ imageSrc, name, className }) {
    const finalClassName = `border border-base-400 rounded-full ${className}`;
    const initials = name ? createInitials(name) : '--';

    return imageSrc ? (
        <img src={imageSrc} alt={initials} className={finalClassName} />
    ) : (
        <span className={`text-xl ${finalClassName}`}>{initials}</span>
    );
}

Avatar.propTypes = {
    imageSrc: PropTypes.string,
    name: PropTypes.string,
    className: PropTypes.string,
};

Avatar.defaultProps = {
    imageSrc: '',
    name: '',
    className: '',
};

export default Avatar;
