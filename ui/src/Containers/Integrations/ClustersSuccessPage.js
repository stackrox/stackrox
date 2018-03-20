import React, { Component } from 'react';
import * as Icon from 'react-feather';

class ClustersSuccessPage extends Component {
    renderSuccessMessage = () => (
        <div className="flex flex-row flex text-success-600 bg-success-200 border border-solid border-success-400 p-4 items-center">
            <div className="flex-1 text-center">
                <Icon.CheckCircle />
            </div>
            <div className="flex-3 pl-2">
                Success! The cluster has been recognized properly by Apollo.You may now save the
                configuration
            </div>
        </div>
    );

    render() {
        return (
            <div className="flex flex-col text-primary-500 pl-4 pr-4">
                <div className="py-4 border-b border-base-300 border-dashed">
                    Reading Configuration
                </div>
                <div className="py-4">Waiting for the cluster to check-in successfully...</div>
                {this.renderSuccessMessage()}
            </div>
        );
    }
}

export default ClustersSuccessPage;
