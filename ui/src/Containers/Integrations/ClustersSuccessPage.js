import React, { Component } from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';
import { ClipLoader } from 'react-spinners';

class ClustersSuccessPage extends Component {
    static propTypes = {
        success: PropTypes.bool
    };

    static defaultProps = {
        success: false
    };

    renderMessage = () =>
        this.props.success ? this.renderSuccessMessage() : this.renderLoadingMessage();

    renderSuccessMessage = () => (
        <div className="flex flex-row flex text-success-600 bg-success-200 border border-solid border-success-400 p-4 items-center">
            <div className="flex-1 text-center">
                <Icon.CheckCircle />
            </div>
            <div className="flex-3 pl-2">
                Success! The cluster has been recognized properly by Prevent.You may now save the
                configuration
            </div>
        </div>
    );

    renderLoadingMessage = () => (
        <div className="flex flex-row flex text-primary-600 bg-primary-200 border border-solid border-primary-400 p-4 items-center">
            <div className="text-center px-4">
                <ClipLoader color="currentColor" loading size={20} />
            </div>
            <div className="flex-3 pl-2">Waiting for the cluster to check-in successfully...</div>
        </div>
    );

    render() {
        return <div className="flex flex-col text-primary-500 p-4">{this.renderMessage()}</div>;
    }
}

export default ClustersSuccessPage;
