import React, { ReactElement, useEffect, useState } from 'react';
import { connect } from 'react-redux';

import { actions as graphActions } from 'reducers/network/graph';
import ToggleSwitch from 'Components/ToggleSwitch';
import { getHideDefaultExternalSrcs, setHideDefaultExternalSrcs } from 'services/NetworkService';

const DefaultCIDRToggle = ({ updateNetworkNodes }): ReactElement => {
    const [showDefaultExternalSrcs, setShowDefaultExternalSrcs] = useState<boolean>(false);
    const [errorMessage, setErrorMessage] = useState<string>();

    useEffect(() => {
        getHideDefaultExternalSrcs().then(({ response }) => {
            setShowDefaultExternalSrcs(!response.hideDefaultExternalSrcs);
        });
    }, [setShowDefaultExternalSrcs]);

    function toggleHandler(): void {
        setHideDefaultExternalSrcs(showDefaultExternalSrcs)
            .then(() => {
                setShowDefaultExternalSrcs(!showDefaultExternalSrcs);
                updateNetworkNodes();
            })
            .catch(({ message }) => {
                setErrorMessage(message);
            });
    }

    return (
        <div className="border border-base-400 flex items-center justify-between m-4 p-2 rounded">
            Display auto-discovered CIDR blocks in Network Graph
            <span className="text-alert-500">{errorMessage}</span>
            <ToggleSwitch
                toggleHandler={toggleHandler}
                id="default-cidr-toggle"
                enabled={showDefaultExternalSrcs}
            />
        </div>
    );
};

const mapDispatchToProps = {
    updateNetworkNodes: graphActions.updateNetworkNodes,
};

export default connect(null, mapDispatchToProps)(DefaultCIDRToggle);
