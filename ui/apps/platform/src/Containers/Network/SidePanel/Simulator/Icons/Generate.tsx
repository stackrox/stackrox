import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { Tooltip } from '@patternfly/react-core';

import { actions as sidepanelActions } from 'reducers/network/sidepanel';
import generate from 'images/generate.svg';

type GenerateProps = {
    generatePolicyModification: () => void;
};

function Generate({ generatePolicyModification }: GenerateProps): ReactElement {
    function onClick() {
        generatePolicyModification();
    }

    return (
        <Tooltip content="Generate a new YAML">
            <button
                type="button"
                className="inline-block px-2 py-2 border-r border-base-300 cursor-pointer"
                onClick={onClick}
            >
                <img
                    className="text-primary-700 h-4 w-4 hover:bg-base-200"
                    alt="generate"
                    src={generate}
                />
            </button>
        </Tooltip>
    );
}

const mapDispatchToProps = {
    generatePolicyModification: sidepanelActions.generateNetworkPolicyModification,
};

export default connect(null, mapDispatchToProps)(Generate);
