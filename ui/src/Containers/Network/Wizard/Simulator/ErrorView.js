import React, { Component } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import PropTypes from 'prop-types';
import Message from 'Components/Message';

import { selectors } from 'reducers';

class ErrorView extends Component {
    static propTypes = {
        modificationName: PropTypes.string,
        modification: PropTypes.shape({
            applyYaml: PropTypes.string.isRequired
        }),
        modificationState: PropTypes.string.isRequired,
        policyGraphState: PropTypes.string.isRequired
    };

    static defaultProps = {
        modificationName: 'YAML',
        modification: null
    };

    renderYamlFile = () => {
        if (!this.props.modification) {
            return null;
        }

        const { applyYaml } = this.props.modification;
        if (!applyYaml) {
            return null;
        }
        return (
            <div className="flex flex-1 flex-col bg-base-100 relative h-full">
                <div className="border-b border-base-300 p-3 text-base-600 font-700">
                    {this.props.modificationName}
                </div>
                <div className="overflow-auto p-3">
                    <pre className="leading-tight whitespace-pre-wrap word-break">{applyYaml}</pre>
                </div>
            </div>
        );
    };

    render() {
        const { modificationState, policyGraphState } = this.props;
        if (modificationState !== 'ERROR' && policyGraphState !== 'ERROR') return null;

        let errorMessage = '';
        if (modificationState === 'ERROR') {
            errorMessage = 'Unable to generate network policies.';
        } else {
            errorMessage = 'Unable to simulate network policies.';
        }

        return (
            <div className="flex flex-col flex-1">
                <section className="bg-base-100 flex flex-col shadow text-base-600 border border-base-200 m-3 mt-4 mb-4 overflow-hidden h-full">
                    <Message type="error" message={errorMessage} />
                    {this.renderYamlFile()}
                </section>
            </div>
        );
    }
}

const mapStateToProps = createStructuredSelector({
    modification: selectors.getNetworkPolicyModification,
    modificationName: selectors.getNetworkPolicyModificationName,
    modificationState: selectors.getNetworkPolicyModificationState,
    policyGraphState: selectors.getNetworkPolicyGraphState
});

export default connect(mapStateToProps)(ErrorView);
