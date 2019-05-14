import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';

import { ToastContainer, toast } from 'react-toastify';

class Notifications extends Component {
    static propTypes = {
        notifications: PropTypes.arrayOf(PropTypes.string)
    };

    static defaultProps = {
        notifications: []
    };

    showLatestToast = () => {
        if (this.props.notifications[0]) toast(this.props.notifications[0]);
    };

    render() {
        return (
            <ToastContainer
                toastClassName="font-sans text-base-600 text-base-100 font-600 bg-base-100"
                hideProgressBar
                autoClose={3000}
            >
                {this.showLatestToast()}
            </ToastContainer>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    notifications: selectors.getNotifications
});

export default connect(mapStateToProps)(Notifications);
