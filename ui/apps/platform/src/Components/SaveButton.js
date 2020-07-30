import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { submit, isPristine, isValid } from 'redux-form';

const onClickHandler = (dispatch, formName) => () => {
    dispatch(submit(formName));
};

const SaveButton = ({ dispatch, formName, className, isPristineForm, isValidForm }) => (
    <button
        className={`btn btn-success ${className}`}
        type="button"
        disabled={isPristineForm || !isValidForm}
        onClick={onClickHandler(dispatch, formName)}
        data-testid="save-btn"
    >
        Save
    </button>
);

SaveButton.propTypes = {
    dispatch: PropTypes.func.isRequired,
    formName: PropTypes.string.isRequired,
    className: PropTypes.string,
    isPristineForm: PropTypes.bool,
    isValidForm: PropTypes.bool,
};

SaveButton.defaultProps = {
    className: '',
    isPristineForm: false,
    isValidForm: true,
};

const mapStateToProps = (state, props) => ({
    isPristineForm: isPristine(props.formName)(state),
    isValidForm: isValid(props.formName)(state),
});

export default connect(mapStateToProps)(SaveButton);
