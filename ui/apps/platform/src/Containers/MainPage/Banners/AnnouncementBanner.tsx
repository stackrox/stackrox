import React, { ReactElement, useEffect, useState } from 'react';
import { AlertVariant, Banner, Button } from '@patternfly/react-core';

import { fetchDatabaseStatus } from 'services/DatabaseService';

const ANNOUNCEMENT_BANNER_KEY = 'postgresAnnouncementBannerDismissed';

function AnnouncementBanner(): ReactElement | null {
    const [isDisplayed, setIsDisplayed] = useState(false);
    const [databaseType, setDatabaseType] = useState('');

    useEffect(() => {
        const localStorageValue = localStorage.getItem(ANNOUNCEMENT_BANNER_KEY);
        const isBannerDismissed = localStorageValue
            ? Boolean(JSON.parse(localStorageValue))
            : false;
        setIsDisplayed(!isBannerDismissed);

        if (!isBannerDismissed) {
            fetchDatabaseStatus()
                .then((response) => {
                    setDatabaseType(response?.databaseType || '');
                })
                .catch(() => {
                    setDatabaseType('');
                });
        }
    }, []);

    function handleDismissClick() {
        localStorage.setItem(ANNOUNCEMENT_BANNER_KEY, JSON.stringify(true));
        setIsDisplayed(false);
    }

    if (isDisplayed && databaseType !== 'PostgresDB') {
        return (
            <Banner
                className="pf-u-display-flex pf-u-justify-content-center pf-u-align-items-center"
                variant={AlertVariant.info}
                style={{ whiteSpace: 'normal' }}
            >
                <span className="pf-u-text-align-center">
                    Red Hat Advanced Cluster Security plans to change its database to PostgreSQL in
                    an upcoming major release. This change will require you to back up your database
                    before upgrading.
                </span>
                <Button className="pf-u-ml-md" onClick={handleDismissClick} variant="link" isInline>
                    dismiss
                </Button>
            </Banner>
        );
    }
    return null;
}

export default AnnouncementBanner;
