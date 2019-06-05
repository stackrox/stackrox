import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { submit, isPristine } from 'redux-form';

const onClickHandler = (dispatch, formName) => () => {
    dispatch(submit(formName));
};

const SaveButton = ({ dispatch, formName, className, isPristineForm }) => (
    <button
        className={`btn btn-success ${className}`}
        type="button"
        disabled={isPristineForm}
        onClick={onClickHandler(dispatch, formName)}
        data-test-id="save-btn"
    >
        Save
    </button>
);

SaveButton.propTypes = {
    dispatch: PropTypes.func.isRequired,
    formName: PropTypes.string.isRequired,
    className: PropTypes.string,
    isPristineForm: PropTypes.bool
};

SaveButton.defaultProps = {
    className: '',
    isPristineForm: false
};

const mapStateToProps = (state, props) => ({ isPristineForm: isPristine(props.formName)(state) });

export default connect(mapStateToProps)(SaveButton);
