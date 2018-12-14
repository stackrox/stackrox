import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { submit } from 'redux-form';

const onClickHandler = (dispatch, formName) => () => {
    dispatch(submit(formName));
};

const SaveButton = ({ dispatch, formName, className }) => (
    <button
        className={`btn btn-success ${className}`}
        type="button"
        onClick={onClickHandler(dispatch, formName)}
    >
        Save
    </button>
);

SaveButton.propTypes = {
    dispatch: PropTypes.func.isRequired,
    formName: PropTypes.string.isRequired,
    className: PropTypes.string
};

SaveButton.defaultProps = {
    className: ''
};

export default connect()(SaveButton);
