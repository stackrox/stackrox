import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';

import { selectors } from 'reducers';
import { actions as dialogueActions } from 'reducers/network/dialogue';
import dialogueStages from 'Containers/Network/Dialogue/dialogueStages';
import { NetworkPolicyModification } from 'Containers/Network/networkTypes';
import Button from './Button';

type NotifyProps = {
    modification: NetworkPolicyModification;
    notifiers?: Record<string, unknown>[];
    setDialogueStage: (stage) => void;
};

function Notify({ modification, notifiers = [], setDialogueStage }: NotifyProps): ReactElement {
    function onClick() {
        setDialogueStage(dialogueStages.notification);
    }

    const noNotifiers = notifiers.length === 0;

    const { applyYaml, toDelete } = modification;
    const noModification = applyYaml === '' && (!toDelete || toDelete.length === 0);
    return (
        <div className="ml-3">
            <Button
                dataTestId="share-yaml-btn"
                onClick={onClick}
                disabled={noNotifiers || noModification}
                icon={<Icon.Share2 className="h-4 w-4 mr-2" />}
                text="Share YAML"
            />
        </div>
    );
}

const mapStateToProps = createStructuredSelector({
    modification: selectors.getNetworkPolicyModification,
    notifiers: selectors.getNotifiers,
});

const mapDispatchToProps = {
    setDialogueStage: dialogueActions.setNetworkDialogueStage,
};

export default connect(mapStateToProps, mapDispatchToProps)(Notify);
