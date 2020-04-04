import React from 'react';
import PropTypes from 'prop-types';

import PolicyBuilderKey from 'Components/PolicyBuilderKey';

function PolicyBuilderKeys({ keys, className }) {
    return (
        <div className={`flex flex-col ${className}`}>
            {keys.map(key => {
                return (
                    <PolicyBuilderKey
                        key={key.jsonpath}
                        label={key.label}
                        jsonpath={key.jsonpath}
                    />
                );
            })}
        </div>
    );
}

PolicyBuilderKeys.propTypes = {
    className: PropTypes.string,
    keys: PropTypes.arrayOf(PropTypes.shape({})).isRequired
};

PolicyBuilderKeys.defaultProps = {
    className: 'w-1/3'
};

export default PolicyBuilderKeys;
