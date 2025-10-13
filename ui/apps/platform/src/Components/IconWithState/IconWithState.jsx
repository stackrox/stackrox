import React from 'react';
import PropTypes from 'prop-types';

function IconWithState({ Icon, enabled }) {
    const wrapperClassName = `flex justify-center items-center h-4 w-4 m-1 rounded-lg ${
        enabled ? 'bg-success-300' : 'bg-base-200'
    }`;
    const iconClassName = `h-2 w-2 ${enabled ? 'text-success-700' : 'text-base-500'}`;

    return (
        <div className={wrapperClassName}>
            <Icon className={iconClassName} />
        </div>
    );
}

IconWithState.propTypes = {
    Icon: PropTypes.elementType.isRequired,
    enabled: PropTypes.bool,
};

IconWithState.defaultProps = {
    enabled: false,
};

export default IconWithState;
