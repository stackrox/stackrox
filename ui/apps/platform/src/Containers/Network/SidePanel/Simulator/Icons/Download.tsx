import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { Tooltip } from '@patternfly/react-core';

import { selectors } from 'reducers';
import download from 'utils/download';
import { NetworkPolicyModification } from 'Containers/Network/networkTypes';

type DownloadProps = {
    modificationName: string;
    modification: NetworkPolicyModification;
};

function Download({ modification, modificationName }: DownloadProps): ReactElement {
    function onClick() {
        const { applyYaml } = modification;
        const formattedYaml = applyYaml.split('\\n').join('\n');

        const yamlName = modificationName.split(/.yaml|.yml/g)[0];
        download(`${yamlName}.yaml`, formattedYaml, 'yaml');
    }

    return (
        <Tooltip content="Download YAML">
            <button
                type="button"
                className="inline-block px-2 py-2 border-base-300 cursor-pointer"
                onClick={onClick}
            >
                <Icon.Download className="h-4 w-4 text-base-500 hover:bg-base-200" />
            </button>
        </Tooltip>
    );
}

const mapStateToProps = createStructuredSelector({
    modificationName: selectors.getNetworkPolicyModificationName,
    modification: selectors.getNetworkPolicyModification,
});

export default connect(mapStateToProps)(Download);
