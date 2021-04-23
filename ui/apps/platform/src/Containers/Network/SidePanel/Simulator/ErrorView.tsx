import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Message } from '@stackrox/ui-components';

import { selectors } from 'reducers';
import { NetworkPolicyModification } from 'Containers/Network/networkTypes';

type ErrorViewProps = {
    modificationName?: string;
    modification?: NetworkPolicyModification;
    modificationState: string;
};

function ErrorView({
    modification,
    modificationName = 'YAML',
    modificationState,
}: ErrorViewProps): ReactElement {
    let errorMessage = '';
    if (modificationState === 'ERROR') {
        errorMessage = 'Unable to generate network policies.';
    } else {
        errorMessage = 'Unable to simulate network policies.';
    }

    return (
        <div className="flex flex-col flex-1">
            <section className="bg-base-100 flex flex-col shadow text-base-600 border border-base-200 m-3 mt-4 mb-4 overflow-hidden h-full">
                <Message type="error">{errorMessage}</Message>
                {modification?.applyYaml && (
                    <div className="flex flex-1 flex-col bg-base-100 relative h-full">
                        <div className="border-b border-base-300 p-3 text-base-600 font-700">
                            {modificationName}
                        </div>
                        <div className="overflow-auto p-3">
                            <pre className="leading-tight whitespace-pre-wrap word-break">
                                {modification.applyYaml}
                            </pre>
                        </div>
                    </div>
                )}
            </section>
        </div>
    );
}

const mapStateToProps = createStructuredSelector({
    modification: selectors.getNetworkPolicyModification,
    modificationName: selectors.getNetworkPolicyModificationName,
    modificationState: selectors.getNetworkPolicyModificationState,
});

export default connect(mapStateToProps)(ErrorView);
