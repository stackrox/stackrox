import React from 'react';
import PropTypes from 'prop-types';
import { Button } from '@patternfly/react-core';

import SaveButton from 'Components/SaveButton';

function FormEditButtons({ formName, isEditing, setIsEditing }) {
    function setEditingFalse() {
        setIsEditing(false);
    }
    function setEditingTrue() {
        setIsEditing(true);
    }
    if (isEditing) {
        return (
            <>
                <Button
                    variant="secondary"
                    className="pf-u-mr-sm"
                    onClick={setEditingFalse}
                    data-testid="cancel-btn"
                >
                    Cancel
                </Button>
                <SaveButton formName={formName} />
            </>
        );
    }
    return (
        <Button variant="primary" data-testid="edit-btn" onClick={setEditingTrue}>
            Edit
        </Button>
    );
}

FormEditButtons.propTypes = {
    formName: PropTypes.string.isRequired,
    isEditing: PropTypes.bool.isRequired,
    setIsEditing: PropTypes.func.isRequired,
};

FormEditButtons.defaultProps = {};

export default FormEditButtons;
