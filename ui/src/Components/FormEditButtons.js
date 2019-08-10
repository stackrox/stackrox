import React from 'react';
import SaveButton from 'Components/SaveButton';
import PropTypes from 'prop-types';

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
                <button
                    className="btn btn-base mr-2"
                    type="button"
                    onClick={setEditingFalse}
                    data-test-id="cancel-btn"
                >
                    Cancel
                </button>
                <SaveButton formName={formName} />
            </>
        );
    }
    return (
        <button
            data-test-id="edit-btn"
            className="btn btn-base"
            type="button"
            onClick={setEditingTrue}
            disabled={isEditing}
        >
            Edit
        </button>
    );
}

FormEditButtons.propTypes = {
    formName: PropTypes.string.isRequired,
    isEditing: PropTypes.bool.isRequired,
    setIsEditing: PropTypes.func.isRequired
};

FormEditButtons.defaultProps = {};

export default FormEditButtons;
