import React from 'react';
import PropTypes from 'prop-types';
import ScopeArray from './ScopeArray';

const WhitelistScope = ({ fields }) => {
    return <ScopeArray fields={fields} isDeploymentScope />;
};

WhitelistScope.propTypes = {
    fields: PropTypes.shape({
        map: PropTypes.func.isRequired,
    }).isRequired,
};

export default WhitelistScope;
