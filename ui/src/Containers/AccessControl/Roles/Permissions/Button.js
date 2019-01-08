import React from 'react';
import PropTypes from 'prop-types';

import SaveButton from 'Containers/AccessControl/SaveButton';

function Button(props) {
    if (!props.isEditing)
        return (
            <button className="btn btn-primary" type="button" onClick={props.onEdit}>
                Edit
            </button>
        );
    return (
        <div className="flex flex-row">
            <button className="btn btn-primary mr-2" type="button" onClick={props.onCancel}>
                Cancel
            </button>
            <SaveButton formName="role-form" />
        </div>
    );
}

Button.propTypes = {
    isEditing: PropTypes.bool.isRequired,
    onEdit: PropTypes.func.isRequired,
    onCancel: PropTypes.func.isRequired
};

export default Button;
