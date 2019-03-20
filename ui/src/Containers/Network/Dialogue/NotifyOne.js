import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

// ConfirmationDialogue is the pop-up that displays when deleting policies from the table.
function NotifyOne(props) {
    if (props.notifiers.length !== 1) {
        return null;
    }
    return (
        <div className="p-3 border-b border-base-300 bg-base-100">
            Notify {props.notifiers[0].name}?
        </div>
    );
}

NotifyOne.propTypes = {
    notifiers: PropTypes.arrayOf(
        PropTypes.shape({
            name: PropTypes.string.isRequired,
            value: PropTypes.string.isRequired
        })
    ).isRequired
};

const mapStateToProps = createStructuredSelector({
    notifiers: selectors.getNotifiers
});

export default connect(mapStateToProps)(NotifyOne);
