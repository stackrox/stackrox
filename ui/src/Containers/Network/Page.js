import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { actions as backendActions } from 'reducers/network/backend';
import { actions as pageActions } from 'reducers/network/page';

import Graph from './Graph/Graph';
import Header from './Header/Header';
import SimulationBorder from './SimulationBorder';
import Wizard from './Wizard/Wizard';

class Page extends Component {
    static propTypes = {
        closeWizard: PropTypes.func.isRequired,
        setNetworkModification: PropTypes.func.isRequired
    };

    componentWillUnmount() {
        this.props.closeWizard();
        this.props.setNetworkModification(null);
    }

    setGraphRef = instance => {
        this.graphInstance = instance;
    };

    getGraphRef = () => this.graphInstance;

    render() {
        return (
            <section className="flex flex-1 h-full w-full">
                <div className="flex flex-1 flex-col w-full">
                    <div className="flex">
                        <Header />
                    </div>
                    <section className="network-grid-bg flex flex-1 relative">
                        <SimulationBorder />
                        <Graph setGraphRef={this.setGraphRef} />
                        <Wizard getGraphRef={this.getGraphRef} />
                    </section>
                </div>
            </section>
        );
    }
}

const mapDispatchToProps = {
    closeWizard: pageActions.closeNetworkWizard,
    setNetworkModification: backendActions.fetchNetworkPolicyModification.success
};

export default connect(
    null,
    mapDispatchToProps
)(Page);
