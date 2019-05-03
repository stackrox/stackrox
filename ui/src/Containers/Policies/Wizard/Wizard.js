import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { selectors } from 'reducers';
import { actions as pageActions } from 'reducers/policies/page';
import { actions as tableActions } from 'reducers/policies/table';
import { createStructuredSelector } from 'reselect';

import Panel from 'Components/Panel';

import Buttons from 'Containers/Policies/Wizard/Buttons';
import WizardPanel from 'Containers/Policies/Wizard/Panel';

// Wizard is the side panel that pops up when you click on a row in the table.
class Wizard extends Component {
    static propTypes = {
        wizardPolicy: PropTypes.shape({
            name: PropTypes.string
        }),
        wizardOpen: PropTypes.bool.isRequired,

        closeWizard: PropTypes.func.isRequired,
        selectPolicyId: PropTypes.func.isRequired
    };

    static defaultProps = {
        wizardPolicy: null
    };

    onClose = () => {
        this.props.closeWizard();
        this.props.selectPolicyId('');
    };

    render() {
        if (!this.props.wizardOpen) return null;

        const header = this.props.wizardPolicy === null ? '' : this.props.wizardPolicy.name;
        return (
            <Panel
                header={header}
                headerComponents={<Buttons />}
                onClose={this.onClose}
                className="bg-primary-200 w-1/2"
            >
                <div className="bg-primary-100 w-full">
                    <div className="h-full bg-base-200">
                        <WizardPanel />
                    </div>
                </div>
            </Panel>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    wizardPolicy: selectors.getWizardPolicy,
    wizardOpen: selectors.getWizardOpen
});

const mapDispatchToProps = {
    closeWizard: pageActions.closeWizard,
    selectPolicyId: tableActions.selectPolicyId
};

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(Wizard);
