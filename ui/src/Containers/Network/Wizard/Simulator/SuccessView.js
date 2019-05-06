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
            applyYaml: PropTypes.string.isRequired,
            toDelete: PropTypes.arrayOf(
                PropTypes.shape({
                    namespace: PropTypes.string.isRequired,
                    name: PropTypes.string.isRequired
                })
            )
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
        const tabs = [{ text: this.props.modificationName }];
        const { applyYaml, toDelete } = this.props.modification;
        const hasToDelete = toDelete && toDelete.length > 0;
        const hasApplyYaml = applyYaml && applyYaml.length >= 2;

        // Format toDelete portion of YAML.
        let toDeleteSection;
        if (hasToDelete) {
            toDeleteSection = toDelete
                .map(entry => `# kubectl -n ${entry.namespace} delete networkpolicy ${entry.name}`)
                .join('\n');
        }

        // Format complete YAML for display.
        let displayYaml;
        if (hasToDelete && hasApplyYaml) {
            displayYaml = [toDeleteSection, applyYaml].join('\n---\n');
        } else if (hasToDelete && !hasApplyYaml) {
            displayYaml = toDeleteSection;
        } else if (!hasToDelete && hasApplyYaml) {
            displayYaml = applyYaml;
        } else {
            displayYaml = 'No policies need to be created or deleted.';
        }

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
            successMessage = 'Policies processed';
        }
        if (modificationSource === 'GENERATED') {
            successMessage = `Policies generated from ${timeWindowMessage}`;
        }
        if (modificationSource === 'ACTIVE') {
            successMessage = 'Viewing active policies';
        }
        if (modificationSource === 'UNDO') {
            successMessage = 'Viewing modification that will undo last applied change';
        }

        return (
            <div className="flex flex-col w-full h-full space-between">
                <section className="flex flex-col bg-base-100 shadow text-base-600 border border-base-200 m-3 mt-4 overflow-hidden h-full">
                    <Message type="success" message={successMessage} />
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
