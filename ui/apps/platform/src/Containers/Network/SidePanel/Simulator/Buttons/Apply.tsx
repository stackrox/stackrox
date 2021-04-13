import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';

import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import Button from './Button';

type ApplyProps = {
    modification: {
        applyYaml: string;
        toDelete: {
            namespace: string;
            name: string;
        }[];
    };
    applicationState: string;
    setDialogueStage: (stage) => void;
};

function Apply({ modification, applicationState, setDialogueStage }: ApplyProps): ReactElement {
    function onClick() {
        setDialogueStage(dialogueStages.application);
    }

    const inRequest = applicationState === 'REQUEST';
    const { applyYaml, toDelete } = modification;
    const noModification = applyYaml === '' && (!toDelete || toDelete.length === 0);
    return (
        <Button
            onClick={onClick}
            disabled={inRequest || noModification}
            icon={<Icon.Save className="h-4 w-4 mr-2" />}
            text="Apply Network Policies"
            dataTestId="apply-network-policies-btn"
        />
    );
}

const mapStateToProps = createStructuredSelector({
    modification: selectors.getNetworkPolicyModification,
    applicationState: selectors.getNetworkPolicyApplicationState,
});

const mapDispatchToProps = {
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(Apply);
