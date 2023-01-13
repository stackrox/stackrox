import React, { useState, useEffect, ReactElement } from 'react';
import { AlertVariant, Banner } from '@patternfly/react-core';

import useInterval from 'hooks/useInterval';
import { fetchDatabaseStatus } from 'services/DatabaseService';

export type DatabaseBannerProps = {
    isApiReachable: boolean;
};

function DatabaseBanner({ isApiReachable }: DatabaseBannerProps): ReactElement | null {
    // To handle database status refreshing.
    const [pollEpoch, setPollEpoch] = useState(0);
    const [databaseAvailable, setDatabaseAvailable] = useState(true);

    // We will update the poll epoch after 60 seconds to force a refresh of the database status
    useInterval(() => {
        setPollEpoch(pollEpoch + 1);
    }, 60000);

    useEffect(() => {
        fetchDatabaseStatus()
            .then((response) => {
                setDatabaseAvailable(Boolean(response?.databaseAvailable));
            })
            .catch(() => {
                setDatabaseAvailable(false);
            });
    }, [pollEpoch]);

    const showDatabaseWarning = isApiReachable && !databaseAvailable;

    if (showDatabaseWarning) {
        return (
            <Banner className="pf-u-text-align-center" isSticky variant={AlertVariant.danger}>
                <span className="pf-u-text-align-center">
                    The database is currently not available. If this problem persists, please
                    contact support.
                </span>
            </Banner>
        );
    }
    return null;
}

export default DatabaseBanner;
