import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { Message } from '@stackrox/ui-components';

import { selectors } from 'reducers';
import { NetworkPolicyModification } from 'Containers/Network/networkTypes';
import SuccessViewTabs from './SuccessViewTabs';
import Download from './Icons/Download';
import Generate from './Icons/Generate';
import Undo from './Icons/Undo';
import Upload from './Icons/Upload';
import Apply from './Buttons/Apply';
import Notify from './Buttons/Notify';

type SuccessViewProps = {
    modificationName?: string;
    modificationSource?: string;
    modification?: NetworkPolicyModification;
    timeWindow: string;
};

function SuccessView({
    modificationName = '',
    modification,
    timeWindow,
    modificationSource = 'GENERATED',
}: SuccessViewProps): ReactElement {
    const timeWindowMessage =
        timeWindow === 'All Time'
            ? 'all network activity'
            : `network activity in the ${timeWindow.toLowerCase()}`;

    let successMessage;
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
                <Message type="success">{successMessage}</Message>
                <div className="flex relative h-full border-t border-r border-base-300 flex-1">
                    <SuccessViewTabs
                        modification={modification}
                        modificationName={modificationName}
                    />
                    <div className="absolute right-0 top-0 flex z-10 h-10 items-center">
                        <Undo />
                        <Generate />
                        <Upload />
                        <Download />
                    </div>
                </div>
            </section>
            {modificationSource !== 'ACTIVE' && (
                <div className="flex mt-2 items-center justify-around p-3 bg-primary-200 border-t-2 border-base-100">
                    <Apply />
                    <Notify />
                </div>
            )}
        </div>
    );
}

const mapStateToProps = createStructuredSelector({
    modificationName: selectors.getNetworkPolicyModificationName,
    modificationSource: selectors.getNetworkPolicyModificationSource,
    modification: selectors.getNetworkPolicyModification,
    timeWindow: selectors.getNetworkActivityTimeWindow,
});

export default connect(mapStateToProps)(SuccessView);
