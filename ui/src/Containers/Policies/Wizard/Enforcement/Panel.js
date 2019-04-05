import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';
import intersection from 'lodash/intersection';

import { actions } from 'reducers/policies/wizard';
import Tile from 'Containers/Policies/Wizard/Enforcement/Tile/Tile';
import lifecycleTileMap, {
    lifecycleToEnforcementsMap
} from 'Containers/Policies/Wizard/Enforcement/descriptors';

function enforcementActionsEmpty(enforcementActions) {
    return (
        enforcementActions === undefined ||
        enforcementActions.length === 0 ||
        enforcementActions[0] === 'UNSET_ENFORCEMENT'
    );
}

class Panel extends Component {
    static propTypes = {
        wizardPolicy: PropTypes.shape({
            enforcementActions: PropTypes.arrayOf(PropTypes.string),
            lifecycleStages: PropTypes.arrayOf(PropTypes.string)
        }).isRequired,

        setWizardPolicy: PropTypes.func.isRequired
    };

    // Check that the lifecycle stage for an enforcement type is present.
    lifecycleStageEnabled = stage =>
        intersection(this.props.wizardPolicy.lifecycleStages, [stage]).length > 0;

    // Check if enforcement types are present.
    hasEnforcementForLifecycle = stage =>
        intersection(this.props.wizardPolicy.enforcementActions, lifecycleToEnforcementsMap[stage])
            .length > 0;

    // Add enforcement types.
    addEnforcementsForLifecycle = stage => {
        const newPolicy = Object.assign({}, this.props.wizardPolicy);
        if (enforcementActionsEmpty(newPolicy.enforcementActions)) {
            newPolicy.enforcementActions = [];
        }
        newPolicy.enforcementActions = newPolicy.enforcementActions.concat(
            ...lifecycleToEnforcementsMap[stage]
        );
        this.props.setWizardPolicy(newPolicy);
    };

    // Remove enforcement types.
    removeEnforcementsForLifecycle = stage => {
        if (
            intersection(
                this.props.wizardPolicy.enforcementActions,
                lifecycleToEnforcementsMap[stage]
            ).length > 0
        ) {
            const newPolicy = Object.assign({}, this.props.wizardPolicy);
            newPolicy.enforcementActions = newPolicy.enforcementActions.filter(
                d => !lifecycleToEnforcementsMap[stage].find(v => v === d)
            );
            this.props.setWizardPolicy(newPolicy);
        }
    };

    // Add or remove enforcement actions from the policy being edited (form data).
    toggleStage = stage => () => {
        if (this.hasEnforcementForLifecycle(stage)) {
            this.removeEnforcementsForLifecycle(stage);
        } else {
            this.addEnforcementsForLifecycle(stage);
        }
    };

    render() {
        const lifecycles = Object.keys(lifecycleToEnforcementsMap);
        return (
            <div className="flex flex-col overflow-y-scroll w-full h-1/3 bg-primary-100">
                <h2 className="font-700 flex justify-center pin-t py-4 px-8 sticky text-xs text-base-600 uppercase items-center tracking-wide leading-normal font-700">
                    BASED ON THE FIELDS SELECTED IN YOUR POLICY CONFIGURATION, YOU MAY CHOOSE TO
                    APPLY ENFORCEMENT AT THE FOLLOWING STAGES:
                </h2>
                <div className="border-b border-base-400" />
                <div className="flex flex-col items-center w-full">
                    {lifecycles.map(key => (
                        <Tile
                            key={key}
                            lifecycle={key}
                            enabled={this.lifecycleStageEnabled(key)}
                            applied={this.hasEnforcementForLifecycle(key)}
                            enforcement={lifecycleTileMap[key]}
                            onOffAction={this.toggleStage(key)}
                        />
                    ))}
                </div>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

const mapDispatchToProps = {
    toggleBuildTime: actions.toggleBuildTimeEnforcement,
    toggleDeployTime: actions.toggleDeployTimeEnforcement,
    toggleRunTime: actions.toggleRunTimeEnforcement,

    setWizardPolicy: actions.setWizardPolicy
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Panel);
