import React from 'react';
import PropTypes from 'prop-types';

import Select from 'Components/Select';
import { availableAuthProviders } from 'constants/accessControl';

function AuthProviderSelect(props) {
    const placeholder = 'Add an Auth Provider';
    return (
        <Select
            options={availableAuthProviders}
            placeholder={placeholder}
            onChange={props.onChange}
        />
    );
}

AuthProviderSelect.propTypes = {
    onChange: PropTypes.func.isRequired
};

export default AuthProviderSelect;
