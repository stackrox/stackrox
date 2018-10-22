import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { createStructuredSelector } from 'reselect';
import { actions as wizardActions } from 'reducers/policies/wizard';
import * as Icon from 'react-feather';
import wizardStages from 'Containers/Policies/Wizard/wizardStages';

import PanelButton from 'Components/PanelButton';

class Buttons extends Component {
    static propTypes = {
        wizardPolicyIsNew: PropTypes.bool.isRequired,
        setWizardStage: PropTypes.func.isRequired
    };

    goBackToPreview = () => this.props.setWizardStage(wizardStages.preview);

    onSubmit = () => {
        if (this.props.wizardPolicyIsNew) {
            this.props.setWizardStage(wizardStages.create);
        } else {
            this.props.setWizardStage(wizardStages.save);
        }
    };

    render() {
        return (
            <React.Fragment>
                <PanelButton
                    icon={<Icon.ArrowLeft className="h-4 w-4" />}
                    text="Previous"
                    className="btn btn-primary"
                    onClick={this.goBackToPreview}
                />
                <PanelButton
                    icon={<Icon.Save className="h-4 w-4" />}
                    text="Save"
                    className="btn btn-success"
                    onClick={this.onSubmit}
                />
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

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Buttons);
