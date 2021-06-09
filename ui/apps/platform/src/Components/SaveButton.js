import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { submit, isPristine, isValid } from 'redux-form';
import { Button } from '@patternfly/react-core';

const onClickHandler = (dispatch, formName) => () => {
    dispatch(submit(formName));
};

const SaveButton = ({ dispatch, formName, isPristineForm, isValidForm }) => (
    <Button
        variant="primary"
        disabled={isPristineForm || !isValidForm}
        onClick={onClickHandler(dispatch, formName)}
        data-testid="save-btn"
    >
        Save
    </Button>
);

SaveButton.propTypes = {
    dispatch: PropTypes.func.isRequired,
    formName: PropTypes.string.isRequired,
    isPristineForm: PropTypes.bool,
    isValidForm: PropTypes.bool,
};

SaveButton.defaultProps = {
    isPristineForm: false,
    isValidForm: true,
};

const mapStateToProps = (state, props) => ({
    isPristineForm: isPristine(props.formName)(state),
    isValidForm: isValid(props.formName)(state),
});

export default connect(mapStateToProps)(SaveButton);
