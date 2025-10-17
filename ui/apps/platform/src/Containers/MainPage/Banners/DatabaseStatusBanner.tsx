import { useState, useEffect } from 'react';
import type { ReactElement } from 'react';
import { useSelector } from 'react-redux';
import { Banner } from '@patternfly/react-core';

import useInterval from 'hooks/useInterval';
import { selectors } from 'reducers';
import { fetchDatabaseStatus } from 'services/DatabaseService';

function DatabaseStatusBanner(): ReactElement | null {
    const serverStatus = useSelector(selectors.serverStatusSelector);
    const isServerReachable = serverStatus !== 'UNREACHABLE';

    // To handle database status refreshing.
    const [pollEpoch, setPollEpoch] = useState(0);
    const [isDatabaseAvailable, setIsDatabaseAvailable] = useState(true);

    // We will update the poll epoch after 60 seconds to force a refresh of the database status
    useInterval(() => {
        setPollEpoch(pollEpoch + 1);
    }, 60000);

    useEffect(() => {
        fetchDatabaseStatus()
            .then((response) => {
                setIsDatabaseAvailable(Boolean(response?.databaseAvailable));
            })
            .catch(() => {
                setIsDatabaseAvailable(false);
            });
    }, [pollEpoch]);

    if (isServerReachable && !isDatabaseAvailable) {
        return (
            <Banner className="pf-v5-u-text-align-center" variant={'red'}>
                <span className="pf-v5-u-text-align-center">
                    The database is currently not available. If this problem persists, please
                    contact support.
                </span>
            </Banner>
        );
    }
    return null;
}

export default DatabaseStatusBanner;
