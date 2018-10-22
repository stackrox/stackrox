import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as wizardActions } from 'reducers/policies/wizard';
import { createStructuredSelector } from 'reselect';
import cloneDeep from 'lodash/cloneDeep';
import * as Icon from 'react-feather';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';

import PanelButton from 'Components/PanelButton';

class Buttons extends Component {
    static propTypes = {
        wizardPolicy: PropTypes.shape({}).isRequired,

        setWizardStage: PropTypes.func.isRequired,
        setWizardPolicy: PropTypes.func.isRequired
    };

    goToEdit = () => this.props.setWizardStage(wizardStages.edit);

    onPolicyClone = () => {
        const newPolicy = cloneDeep(this.props.wizardPolicy);
        newPolicy.id = '';
        newPolicy.name += ' (COPY)';
        this.props.setWizardPolicy(newPolicy);

        this.props.setWizardStage(wizardStages.edit);
    };

    render() {
        return (
            <React.Fragment>
                <PanelButton
                    icon={<Icon.Copy className="h-4 w-4" />}
                    text="Clone"
                    className="btn btn-success"
                    onClick={this.onPolicyClone}
                />
                <PanelButton
                    icon={<Icon.Edit className="h-4 w-4" />}
                    text="Edit"
                    className="btn btn-success"
                    onClick={this.goToEdit}
                />
            </React.Fragment>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage,
    setWizardPolicy: wizardActions.setWizardPolicy
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Buttons);
