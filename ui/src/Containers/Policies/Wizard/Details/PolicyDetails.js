import React from 'react';
import PropTypes from 'prop-types';

import Fields from 'Containers/Policies/Wizard/Details/Fields';
import ConfigurationFields from 'Containers/Policies/Wizard/Details/ConfigurationFields';

function PolicyDetails({ policy }) {
    if (!policy) return null;

    return (
        <div className="w-full h-full">
            <div className="flex flex-col w-full overflow-auto pb-5">
                <Fields policy={policy} />
                <ConfigurationFields policy={policy} />
            </div>
        </div>
    );
}

PolicyDetails.propTypes = {
    policy: PropTypes.shape({
        name: PropTypes.string
    }).isRequired
};

export default PolicyDetails;
