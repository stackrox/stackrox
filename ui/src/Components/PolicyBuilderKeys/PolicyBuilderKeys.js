import React from 'react';
import PropTypes from 'prop-types';

import PolicyBuilderKey from 'Components/PolicyBuilderKey';

function PolicyBuilderKeys({ keys, className }) {
    return (
        <div className={`flex flex-col px-3 pt-3 bg-primary-300 ${className}`}>
            <div className="-ml-6 -mr-3 bg-primary-500 mb-2 p-2 rounded-bl rounded-tl text-base-100">
                Drag out a policy field
            </div>
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
