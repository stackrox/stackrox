import React from 'react';
import PropTypes from 'prop-types';

import ScopeArray from './ScopeArray';

function ExcludedScope({ fields }) {
    return <ScopeArray fields={fields} isDeploymentScope />;
}

ExcludedScope.propTypes = {
    fields: PropTypes.shape({
        map: PropTypes.func.isRequired,
    }).isRequired,
};

export default ExcludedScope;
