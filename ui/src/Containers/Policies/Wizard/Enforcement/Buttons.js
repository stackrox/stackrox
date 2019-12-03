import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';

import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { actions as wizardActions } from 'reducers/policies/wizard';
import * as Icon from 'react-feather';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';

import PanelButton from 'Components/PanelButton';

class Buttons extends Component {
    static propTypes = {
        history: ReactRouterPropTypes.location.isRequired,
        match: ReactRouterPropTypes.location.isRequired,
        wizardPolicyIsNew: PropTypes.bool.isRequired,
        setWizardStage: PropTypes.func.isRequired
    };

    goBackToPreview = () => this.props.setWizardStage(wizardStages.preview);

    onSubmit = () => {
        if (this.props.wizardPolicyIsNew) {
            this.props.setWizardStage(wizardStages.create);
        } else {
            this.props.setWizardStage(wizardStages.save);
            this.props.history.push(`/main/policies/${this.props.match.params.policyId}`);
        }
    };

    render() {
        return (
            <React.Fragment>
                <PanelButton
                    icon={<Icon.ArrowLeft className="h-4 w-4" />}
                    className="btn btn-base"
                    onClick={this.goBackToPreview}
                >
                    Previous
                </PanelButton>
                <PanelButton
                    icon={<Icon.Save className="h-4 w-4" />}
                    className="btn btn-success"
                    onClick={this.onSubmit}
                >
                    Save
                </PanelButton>
            </React.Fragment>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    wizardPolicyIsNew: selectors.getWizardIsNew
});

const mapDispatchToProps = {
    setWizardStage: wizardActions.setWizardStage
};

export default withRouter(
    connect(
        mapStateToProps,
        mapDispatchToProps
    )(Buttons)
);
