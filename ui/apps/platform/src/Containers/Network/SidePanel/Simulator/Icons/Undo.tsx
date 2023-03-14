import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import * as Icon from 'react-feather';
import { Tooltip } from '@patternfly/react-core';

import { selectors } from 'reducers';
import { actions as sidepanelActions } from 'reducers/network/sidepanel';

type UndoProps = {
    applicationState: string;
    undoModification: () => void;
};

function Undo({ undoModification, applicationState }: UndoProps): ReactElement {
    function onClick() {
        undoModification();
    }

    return (
        <Tooltip content="Revert most recently applied YAML">
            <button
                type="button"
                disabled={applicationState === 'REQUEST'}
                className="inline-block px-2 py-2 border-l border-r border-base-300 cursor-pointer"
                onClick={onClick}
            >
                <Icon.RotateCcw className="h-4 w-4 text-base-500 hover:bg-base-200" />
            </button>
        </Tooltip>
    );
}

const mapStateToProps = createStructuredSelector({
    applicationState: selectors.getNetworkPolicyApplicationState,
});

const mapDispatchToProps = {
    undoModification: sidepanelActions.loadUndoNetworkPolicyModification,
};

export default connect(mapStateToProps, mapDispatchToProps)(Undo);
