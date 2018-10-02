import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { connect } from 'react-redux';
import { createSelector, createStructuredSelector } from 'reselect';
import { ClipLoader } from 'react-spinners';
import * as Icon from 'react-feather';

import { selectors } from 'reducers';
import { actions } from 'reducers/alerts';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import Panel from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import DeploymentDetails from '../Risk/DeploymentDetails';
import ViolationsDetails from './ViolationsDetails';
import PolicyDetails from '../Policies/PolicyDetails';

class ViolationsPanel extends Component {
    static propTypes = {
        alert: PropTypes.shape({
            id: PropTypes.string
        }),
        whitelistDeployment: PropTypes.func.isRequired,
        onClose: PropTypes.func.isRequired
    };

    static defaultProps = {
        alert: null
    };

    constructor(props) {
        super(props);

        this.state = {
            whitelisting: false
        };
    }

    whitelistDeploymentHandler = () => {
        this.props.whitelistDeployment(this.props.alert);
        this.setState({ whitelisting: true }); // hack: after whitelisting the violation will disappear and the panel will close
    };

    renderTabs = () => {
        const { alert } = this.props;
        const riskPanelTabs = [
            { text: 'Violations' },
            { text: 'Deployment Details' },
            { text: 'Policy Details' }
        ];
        const isLoading = !alert; // TODO: poor-man loading check until a proper one in place
        const content = isLoading ? (
            <div className="flex flex-col items-center justify-center h-full w-full">
                <ClipLoader loading size={20} />
                <div className="text-lg font-sans tracking-wide mt-4">Loading...</div>
            </div>
        ) : (
            <Tabs headers={riskPanelTabs}>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        <ViolationsDetails violations={alert.violations} />
                    </div>
                </TabContent>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        <DeploymentDetails deployment={alert.deployment} />
                    </div>
                </TabContent>
                <TabContent>
                    <div className="flex flex-1 flex-col">
                        <PolicyDetails policy={alert.policy} />
                    </div>
                </TabContent>
            </Tabs>
        );
        return content;
    };

    render() {
        const { alert } = this.props;
        if (!alert || !alert.policy || !alert.deployment) return null; // TODO: show loading

        const header = `${alert.deployment.name} (${alert.deployment.id})`;
        const whitelistButton = (
            <PanelButton
                icon={
                    this.state.whitelisting ? (
                        <ClipLoader color="currentColor" loading size={15} />
                    ) : (
                        <Icon.CheckSquare className="h-4 w-4" />
                    )
                }
                text="Whitelist"
                className="btn btn-success"
                onClick={this.whitelistDeploymentHandler}
                disabled={this.state.whitelisting}
                tooltip="Whitelist deployment for this policy. View whitelists in the policy editing page"
            />
        );
        return (
            <Panel
                header={header}
                buttons={whitelistButton}
                className="w-1/2 bg-primary-200"
                onClose={this.props.onClose}
            >
                {this.renderTabs()}
            </Panel>
        );
    }
}

const getAlert = createSelector(
    [selectors.getFilteredAlerts, (state, props) => props.alertId],
    (alerts, alertId) => alerts.find(alert => alert.id === alertId)
);

const mapStateToProps = createStructuredSelector({
    alert: getAlert
});

const mapDispatchToProps = dispatch => ({
    whitelistDeployment: alert => dispatch(actions.whitelistDeployment.request(alert))
});

export default connect(
    mapStateToProps,
    mapDispatchToProps
)(ViolationsPanel);
