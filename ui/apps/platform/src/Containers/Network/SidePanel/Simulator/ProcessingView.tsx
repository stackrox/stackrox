import React, { ReactElement } from 'react';
import { connect } from 'react-redux';
import { createStructuredSelector } from 'reselect';
import { selectors } from 'reducers';

import Loader from 'Components/Loader';

function ProcessingView(): ReactElement {
    return (
        <div className="flex flex-col flex-1">
            <section className="m-3 flex flex-1 border border-dashed border-base-300 bg-base-100">
                <div className="flex flex-col flex-1 font-500 uppercase">
                    <Loader message="Processing Network Policies" />
                </div>
            </section>
        </div>
    );
}

const mapStateToProps = createStructuredSelector({
    modificationState: selectors.getNetworkPolicyModificationState,
    policyGraphState: selectors.getNetworkPolicyGraphState,
});

export default connect(mapStateToProps)(ProcessingView);
