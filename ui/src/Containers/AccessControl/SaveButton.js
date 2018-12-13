import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { submit } from 'redux-form';

const onClickHandler = (dispatch, formName) => () => {
    dispatch(submit(formName));
};

const SaveButton = ({ dispatch, formName }) => (
    <button className="btn btn-success" type="button" onClick={onClickHandler(dispatch, formName)}>
        Save
    </button>
);

SaveButton.propTypes = {
    dispatch: PropTypes.func.isRequired,
    formName: PropTypes.string.isRequired
};

export default connect()(SaveButton);
