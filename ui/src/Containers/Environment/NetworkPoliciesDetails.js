import React, { Component } from 'react';
import PropTypes from 'prop-types';
import removeEmptyObjects from 'utils/removeEmptyObjects';
import removeEmptyFields from 'utils/removeEmptyFields';

import KeyValuePairs from 'Components/KeyValuePairs';
import CollapsibleCard from 'Components/CollapsibleCard';
import download from 'utils/download';
import * as Icon from 'react-feather';

const networkPolicyDetailsMap = {
    id: { label: 'Policy ID' },
    name: { label: 'Policy Name' },
    clusterName: { label: 'Cluster' },
    namespace: { label: 'Namespace' },
    labels: { label: 'Labels' },
    annotations: { label: 'Annotations' },
    spec: { label: 'Spec' }
};

const processNetworkPoliciesData = data => removeEmptyObjects(removeEmptyFields(data));

class NetworkPoliciesDetails extends Component {
    static propTypes = {
        networkPolicies: PropTypes.arrayOf(PropTypes.shape({ id: PropTypes.string.isRequired }))
            .isRequired
    };

    downloadYamlFile = (name, content, type) => () => {
        download(name, content, type);
    };

    renderOverview() {
        const { networkPolicies } = this.props;
        return (
            <div>
                {networkPolicies.map(networkPolicy => (
                    <div className="px-3 py-4" key={networkPolicy.id}>
                        <CollapsibleCard title={networkPolicy.name}>
                            <div className="h-full pt-3 pl-3 pr-3">
                                <KeyValuePairs
                                    data={processNetworkPoliciesData(networkPolicy)}
                                    keyValueMap={networkPolicyDetailsMap}
                                />
                            </div>
                            <div className="flex justify-center p-3 border-t border-base-300">
                                <button
                                    className="download uppercase text-primary-600 p-2 text-center text-sm border border-solid bg-primary-200 border-primary-300 hover:bg-primary-100"
                                    onClick={this.downloadYamlFile(
                                        `${networkPolicy.name}.yaml`,
                                        networkPolicy.yaml,
                                        'yaml'
                                    )}
                                    tabIndex="-1"
                                >
                                    <span className="pr-2">Download yaml file and keys</span>
                                    <Icon.Download className="h-3 w-3" />
                                </button>
                            </div>
                        </CollapsibleCard>
                    </div>
                ))}
            </div>
        );
    }
    render() {
        return <div className="w-full">{this.renderOverview()}</div>;
    }
}

export default NetworkPoliciesDetails;
