import React from 'react';
import PropTypes from 'prop-types';

import SaveButton from 'Containers/AccessControl/Roles/Permissions/Button/SaveButton';

function Button(props) {
    if (!props.isEditing)
        return (
            <button className="btn btn-primary" type="button" onClick={props.onEdit}>
                Edit
            </button>
        );
    return <SaveButton />;
}

Button.propTypes = {
    isEditing: PropTypes.bool.isRequired,
    onEdit: PropTypes.func.isRequired
};

export default Button;
