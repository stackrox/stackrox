import React, { Component } from 'react';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import DownloadService from 'services/DownloadService';

const baseClass = 'py-4 border-b border-base-300 border-solid';

class ClustersDownloadPage extends Component {
    static propTypes = {
        cluster: PropTypes.shape({
            name: PropTypes.string.isRequired
        }).isRequired
    };

    downloadYamlFile = () => {
        const options = {
            url: '/api/extensions/clusters/zip',
            data: this.props.cluster
        };
        DownloadService(options);
    };
    renderDownloadSection = () => (
        <div className={baseClass}>
            1) Download the required Configuration files
            <div className="flex justify-center pt-3 pb-1">
                <button
                    className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300"
                    onClick={this.downloadYamlFile}
                >
                    <span className="pr-2">Download yaml file and keys</span>
                    <Icon.Download className="h-3 w-3" />
                </button>
            </div>
            <div className="text-xs text-center text-primary-300">
                * You may modify the YAML to suit your environment as needed
            </div>
        </div>
    );

    renderCommandSection = () => (
        <div className={baseClass}>
            3) Run the following command in your CLI
            <div className="flex items-center text-center text-danger-600 bg-base-100 p-4 mt-3">
                docker stack deploy -c agent-$CLUSTER.yaml prevent
            </div>
        </div>
    );

    renderCredentialsText = () => (
        <div className={baseClass}>2) Add your credentials to location X</div>
    );

    renderNextScreenText = () => (
        <div className={baseClass}>
            4) Once the above has been configured, navigate to the next screen to validate and
            confirm that the cluster has been checked in successfully.
        </div>
    );

    render() {
        return (
            <div className="flex flex-col text-primary-500 pl-4 pr-4">
                {this.renderDownloadSection()}
                {this.renderCredentialsText()}
                {this.renderCommandSection()}
                {this.renderNextScreenText()}
            </div>
        );
    }
}

export default ClustersDownloadPage;
