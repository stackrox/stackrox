import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import { selectors } from 'reducers';

import Message from 'Components/Message';
import Tabs from 'Components/Tabs';
import TabContent from 'Components/TabContent';
import UsageButtons from './UsageButtons';
import Download from './Icons/Download';
import Generate from './Icons/Generate';
import Undo from './Icons/Undo';
import Upload from './Icons/Upload';

class SuccessView extends Component {
    static propTypes = {
        modificationName: PropTypes.string,
        modificationSource: PropTypes.string,
        modification: PropTypes.shape({
            applyYaml: PropTypes.string.isRequired
        }),
        modificationState: PropTypes.string.isRequired,
        policyGraphState: PropTypes.string.isRequired,
        timeWindow: PropTypes.string.isRequired
    };

    static defaultProps = {
        modification: null,
        modificationName: '',
        modificationSource: 'GENERATED'
    };

    renderTabs = () => {
        const { applyYaml } = this.props.modification;
        const tabs = [{ text: this.props.modificationName }];
        const displayYaml = applyYaml.length < 2 ? '\n\n(empty policy generated)' : applyYaml;
        return (
            <Tabs headers={tabs}>
                <TabContent>
                    <div className="flex flex-col bg-base-100 overflow-auto h-full">
                        <pre className="p-3 pt-4 leading-tight whitespace-pre-wrap word-break">
                            {displayYaml}
                        </pre>
                    </div>
                </TabContent>
            </Tabs>
        );
    };

    render() {
        const { modification, modificationState, policyGraphState, timeWindow } = this.props;
        if (
            modification === null ||
            modificationState !== 'SUCCESS' ||
            policyGraphState !== 'SUCCESS'
        )
            return null;

        const timeWindowMessage =
            timeWindow === 'All Time'
                ? 'all network activity'
                : `network activity in the ${timeWindow.toLowerCase()}`;

        let successMessage;
        const { modificationSource } = this.props;
        if (modificationSource === 'UPLOAD') {
            successMessage = 'YAML uploaded successfully';
        }
        if (modificationSource === 'GENERATED') {
            successMessage = `YAML generated successfully from ${timeWindowMessage}`;
        }
        if (modificationSource === 'ACTIVE') {
            successMessage = 'Active YAML';
        }
        if (modificationSource === 'UNDO') {
            successMessage = 'Viewing modification that will undo last applied change';
        }

        return (
            <div className="flex flex-col w-full h-full space-between">
                <section className="flex flex-col bg-base-100 shadow text-base-600 border border-base-200 m-3 mt-4 overflow-hidden h-full">
                    <Message type="info" message={successMessage} />
                    <div className="flex relative h-full border-t border-r border-base-300">
                        {this.renderTabs()}
                        <div className="absolute pin-r pin-t h-9 z-10">
                            <Undo />
                            <Generate />
                            <Upload />
                            <Download />
                        </div>
                    </div>
                </section>
                <UsageButtons />
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    modificationName: selectors.getNetworkPolicyModificationName,
    modificationSource: selectors.getNetworkPolicyModificationSource,
    modification: selectors.getNetworkPolicyModification,
    modificationState: selectors.getNetworkPolicyModificationState,
    policyGraphState: selectors.getNetworkPolicyGraphState,
    timeWindow: selectors.getNetworkActivityTimeWindow
});

export default connect(mapStateToProps)(SuccessView);
