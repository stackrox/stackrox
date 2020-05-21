import React from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import intersection from 'lodash/intersection';

import { selectors } from 'reducers';
import Panel from 'Components/Panel';
import { actions } from 'reducers/policies/wizard';
import Tile from 'Containers/Policies/Wizard/Enforcement/Tile/Tile';
import lifecycleTileMap, {
    lifecycleToEnforcementsMap,
} from 'Containers/Policies/Wizard/Enforcement/descriptors';
import EnforcementButtons from 'Containers/Policies/Wizard/Enforcement/EnforcementButtons';

function enforcementActionsEmpty(enforcementActions) {
    return (
        enforcementActions === undefined ||
        enforcementActions.length === 0 ||
        enforcementActions[0] === 'UNSET_ENFORCEMENT'
    );
}

function EnforcementPanel({ header, onClose, wizardPolicy, setWizardPolicy }) {
    // Check that the lifecycle stage for an enforcement type is present.
    function lifecycleStageEnabled(stage) {
        return intersection(wizardPolicy.lifecycleStages, [stage]).length > 0;
    }

    // Check if enforcement types are present.
    function hasEnforcementForLifecycle(stage) {
        return (
            intersection(wizardPolicy.enforcementActions, lifecycleToEnforcementsMap[stage])
                .length > 0
        );
    }

    // Add enforcement types.
    function addEnforcementsForLifecycle(stage) {
        const newPolicy = { ...wizardPolicy };
        if (enforcementActionsEmpty(newPolicy.enforcementActions)) {
            newPolicy.enforcementActions = [];
        }
        newPolicy.enforcementActions = newPolicy.enforcementActions.concat(
            ...lifecycleToEnforcementsMap[stage]
        );
        setWizardPolicy(newPolicy);
    }

    // Remove enforcement types.
    function removeEnforcementsForLifecycle(stage) {
        if (
            intersection(wizardPolicy.enforcementActions, lifecycleToEnforcementsMap[stage])
                .length > 0
        ) {
            const newPolicy = { ...wizardPolicy };
            newPolicy.enforcementActions = newPolicy.enforcementActions.filter(
                (d) => !lifecycleToEnforcementsMap[stage].find((v) => v === d)
            );
            setWizardPolicy(newPolicy);
        }
    }

    // Add or remove enforcement actions from the policy being edited (form data).
    function toggleOn(stage) {
        return () => {
            if (!hasEnforcementForLifecycle(stage)) {
                addEnforcementsForLifecycle(stage);
            }
        };
    }

    function toggleOff(stage) {
        return () => {
            if (hasEnforcementForLifecycle(stage)) {
                removeEnforcementsForLifecycle(stage);
            }
        };
    }

    const lifecycles = Object.keys(lifecycleToEnforcementsMap);
    return (
        <Panel
            header={header}
            headerComponents={<EnforcementButtons />}
            onClose={onClose}
            id="side-panel"
            className="w-1/2"
        >
            <div className="flex flex-col overflow-y-scroll w-full h-1/3 bg-primary-100">
                <h2 className="font-700 flex justify-center top-0 py-4 px-8 sticky text-xs text-base-600 uppercase items-center tracking-wide leading-normal font-700">
                    BASED ON THE FIELDS SELECTED IN YOUR POLICY CONFIGURATION, YOU MAY CHOOSE TO
                    APPLY ENFORCEMENT AT THE FOLLOWING STAGES:
                </h2>
                <div className="border-b border-base-400" />
                <div className="flex flex-col items-center w-full">
                    {lifecycles.map((key) => (
                        <Tile
                            key={key}
                            lifecycle={key}
                            enabled={lifecycleStageEnabled(key)}
                            applied={hasEnforcementForLifecycle(key)}
                            enforcement={lifecycleTileMap[key]}
                            onAction={toggleOn(key)}
                            offAction={toggleOff(key)}
                        />
                    ))}
                </div>
            </div>
        </Panel>
    );
}

EnforcementPanel.propTypes = {
    header: PropTypes.string,
    wizardPolicy: PropTypes.shape({
        enforcementActions: PropTypes.arrayOf(PropTypes.string),
        lifecycleStages: PropTypes.arrayOf(PropTypes.string),
    }).isRequired,

    setWizardPolicy: PropTypes.func.isRequired,
    onClose: PropTypes.func.isRequired,
};

EnforcementPanel.defaultProps = {
    header: '',
};

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy,
});

const mapDispatchToProps = {
    toggleBuildTime: actions.toggleBuildTimeEnforcement,
    toggleDeployTime: actions.toggleDeployTimeEnforcement,
    toggleRunTime: actions.toggleRunTimeEnforcement,

    setWizardPolicy: actions.setWizardPolicy,
};

export default connect(mapStateToProps, mapDispatchToProps)(EnforcementPanel);
