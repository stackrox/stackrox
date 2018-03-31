import React, { Component } from 'react';
import * as Icon from 'react-feather';
import PropTypes from 'prop-types';
import DownloadService from 'services/DownloadService';

const baseClass = 'py-6 border-b border-base-300 border-solid';

class ClustersDownloadPage extends Component {
    static propTypes = {
        onClick: PropTypes.func.isRequired
    };

    downloadYamlFile = clusterId => {
        const options = {
            url: '/api/extensions/clusters/zip',
            data: { id: clusterId }
        };
        DownloadService(options);
    };

    render() {
        return (
            <div className="px-4">
                <div className={baseClass}>
                    1) Download the required Configuration files
                    <div className="flex justify-center p-3">
                        <button
                            className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300 hover:bg-primary-100"
                            onClick={this.props.onClick}
                            tabIndex="-1"
                        >
                            <span className="pr-2">Download yaml file and keys</span>
                            <Icon.Download className="h-3 w-3" />
                        </button>
                    </div>
                    <div className="text-xs text-center text-base-400">
                        * You may modify the YAML to suit your environment as needed
                    </div>
                </div>
                <div className={baseClass}>
                    2) Use the deploy script inside the zip file to launch the sensor in your
                    environment
                </div>
            </div>
        );
    }
}

export default ClustersDownloadPage;
