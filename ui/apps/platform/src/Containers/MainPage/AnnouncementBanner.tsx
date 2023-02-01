import React, { ReactElement, useEffect, useState } from 'react';
import { AlertVariant, Banner, Button } from '@patternfly/react-core';

const ANNOUNCEMENT_BANNER_KEY = 'postgresAnnouncementBannerDismissed';

function AnnouncementBanner(): ReactElement | null {
    const [isDisplayed, setIsDisplayed] = useState(false);

    useEffect(() => {
        const localStorageValue = localStorage.getItem(ANNOUNCEMENT_BANNER_KEY);
        const isBannerDismissed = localStorageValue
            ? Boolean(JSON.parse(localStorageValue))
            : false;
        setIsDisplayed(!isBannerDismissed);
    }, []);

    function handleDismissClick() {
        localStorage.setItem(ANNOUNCEMENT_BANNER_KEY, JSON.stringify(true));
        setIsDisplayed(false);
    }

    if (isDisplayed) {
        return (
            <Banner
                className="pf-u-display-flex pf-u-justify-content-center pf-u-align-items-center"
                isSticky
                variant={AlertVariant.info}
            >
                <span className="pf-u-text-align-center">
                    The next version of this product will be version 4.0.0. Central will be using
                    Postgres for its data store starting in v4.0.0. You must backup your database
                    before upgrading to 4.0.0.
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
