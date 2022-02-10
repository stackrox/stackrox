/* eslint-disable jest/no-disabled-tests */
import React from 'react';
import { render, screen } from '@testing-library/react';
import { format } from 'date-fns';

import VulnRequestedAction from './VulnRequestedAction';

const expiresOnFormat = 'YYYY-MM-DD[T]HH:mm:ss.SSSSSSSSS[Z]';

describe('VulnRequestedAction', () => {
    it('should show requested action for false positives', () => {
        render(
            <VulnRequestedAction
                targetState="FALSE_POSITIVE"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: false,
                    expiresOn: '',
                }}
                currentDate={new Date()}
            />
        );
        expect(screen.getByText('False positive')).toBeDefined();
    });

    it('should show the requested action for an until fixed deferral', () => {
        render(
            <VulnRequestedAction
                targetState="DEFERRED"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: true,
                    expiresOn: '',
                }}
                currentDate={new Date()}
            />
        );
        expect(screen.getByText('Deferral (until fixed)')).toBeDefined();
    });

    it('should show the requested action for a 2 week deferral', () => {
        const currentDate = new Date();
        const expiresOnDate = new Date();
        const daysToAdd = 14;
        expiresOnDate.setDate(expiresOnDate.getDate() + daysToAdd);
        const expiresOn = format(expiresOnDate, expiresOnFormat);

        render(
            <VulnRequestedAction
                targetState="DEFERRED"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: false,
                    expiresOn,
                }}
                currentDate={currentDate}
            />
        );
        expect(screen.getByText('Deferral (14 days)')).toBeDefined();
    });

    it('should show the requested action for a 30 day deferral', () => {
        const currentDate = new Date();
        const expiresOnDate = new Date();
        const daysToAdd = 30;
        expiresOnDate.setDate(currentDate.getDate() + daysToAdd);
        const expiresOn = format(expiresOnDate, expiresOnFormat);

        render(
            <VulnRequestedAction
                targetState="DEFERRED"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: false,
                    expiresOn,
                }}
                currentDate={currentDate}
            />
        );
        expect(screen.getByText('Deferral (30 days)')).toBeDefined();
    });

    it('should show the requested action for a 90 day deferral', () => {
        const currentDate = new Date();
        const expiresOnDate = new Date();
        const daysToAdd = 90;
        expiresOnDate.setDate(currentDate.getDate() + daysToAdd);
        const expiresOn = format(expiresOnDate, expiresOnFormat);

        render(
            <VulnRequestedAction
                targetState="DEFERRED"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: false,
                    expiresOn,
                }}
                currentDate={currentDate}
            />
        );
        expect(screen.getByText('Deferral (90 days)')).toBeDefined();
    });

    // @TODO: Add test for indefinite deferral
});
