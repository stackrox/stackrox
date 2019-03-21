import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions as dialogueActions } from 'reducers/network/dialogue';
import { actions as wizardActions } from 'reducers/network/wizard';
import { actions as pageActions } from 'reducers/network/page';
import dialogueStages from './Dialogue/dialogueStages';

import Dialogue from './Dialogue';
import Graph from './Graph/Graph';
import Header from './Header/Header';
import SimulationBorder from './SimulationBorder';
import Wizard from './Wizard/Wizard';

class Page extends Component {
    static propTypes = {
        setNetworkModification: PropTypes.func.isRequired,
        closeWizard: PropTypes.func.isRequired,
        setDialogueStage: PropTypes.func.isRequired
    };

    componentWillUnmount() {
        this.props.closeWizard();
        this.props.setDialogueStage(dialogueStages.closed);
        this.props.setNetworkModification(null);
    }

    render() {
        return (
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full">
                    <div className="flex">
                        <Header />
                    </div>
                    <section className="network-grid-bg flex flex-1 relative">
                        <SimulationBorder />
                        <Graph />
                        <Wizard />
                    </section>
                </div>
                <Dialogue />
            </section>
        );
    }
}

const mapDispatchToProps = {
    closeWizard: pageActions.closeNetworkWizard,
    setNetworkModification: wizardActions.setNetworkPolicyModification,
    setDialogueStage: dialogueActions.setNetworkDialogueStage
};

export default connect(
    null,
    mapDispatchToProps
)(Page);
