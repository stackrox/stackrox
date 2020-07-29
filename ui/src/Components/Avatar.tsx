import React, { ReactElement } from 'react';
import PropTypes from 'prop-types';
import { find as createInitials } from 'initials';

type Props = {
    imageSrc?: string;
    name?: string;
    className?: string;
};

function Avatar({ imageSrc, name, className = '' }: Props): ReactElement {
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
    imageSrc: undefined,
    name: undefined,
    className: '',
};

export default Avatar;
