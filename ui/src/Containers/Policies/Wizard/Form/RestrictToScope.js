import React from 'react';
import PropTypes from 'prop-types';
import ScopeArray from './ScopeArray';

const RestrictToScope = ({ fields }) => {
    return <ScopeArray fields={fields} isDeploymentScope={false} />;
};

RestrictToScope.propTypes = {
    fields: PropTypes.shape({
        map: PropTypes.func.isRequired
    }).isRequired
};

export default RestrictToScope;
