import React, { ReactElement, useEffect, useState } from 'react';
import { Alert, Flex, Switch } from '@patternfly/react-core';

import { getHideDefaultExternalSrcs, setHideDefaultExternalSrcs } from 'services/NetworkService';

function DefaultCIDRToggle({ updateNetworkNodes = () => {} }): ReactElement {
    const [showDefaultExternalSrcs, setShowDefaultExternalSrcs] = useState(false);
    const [errorMessage, setErrorMessage] = useState('');

    useEffect(() => {
        getHideDefaultExternalSrcs()
            .then(({ response }) => {
                setShowDefaultExternalSrcs(!response.hideDefaultExternalSrcs);
                setErrorMessage('');
            })
            .catch(({ message }) => {
                setErrorMessage(message);
            });
        return () => {
            setShowDefaultExternalSrcs(false);
            setErrorMessage('');
        };
    }, [setShowDefaultExternalSrcs]);

    function toggleHandler(): void {
        setHideDefaultExternalSrcs(showDefaultExternalSrcs)
            .then(() => {
                setShowDefaultExternalSrcs(!showDefaultExternalSrcs);
                setErrorMessage('');
                updateNetworkNodes();
            })
            .catch(({ message }) => {
                setErrorMessage(message);
            });
    }

    return (
        <Flex className="pf-u-mb-md">
            <Switch
                id="default-cidr-toggle"
                isChecked={showDefaultExternalSrcs}
                onChange={toggleHandler}
                label="Auto-discovered CIDR blocks"
            />
            {errorMessage && <Alert variant="danger" title={errorMessage} />}
        </Flex>
    );
}

export default DefaultCIDRToggle;
