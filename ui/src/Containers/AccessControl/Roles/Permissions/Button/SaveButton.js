import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { submit } from 'redux-form';

const onClickHandler = dispatch => () => {
    dispatch(submit('role-form'));
};

const SaveButton = ({ dispatch }) => (
    <button className="btn btn-success" type="button" onClick={onClickHandler(dispatch)}>
        Save
    </button>
);

SaveButton.propTypes = {
    dispatch: PropTypes.func.isRequired
};

export default connect()(SaveButton);
