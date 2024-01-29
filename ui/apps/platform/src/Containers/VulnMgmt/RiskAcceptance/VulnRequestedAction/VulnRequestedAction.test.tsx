/* eslint-disable jest/no-disabled-tests */
import React from 'react';
import { render, screen } from '@testing-library/react';
import { addDays } from 'date-fns';

import VulnRequestedAction from './VulnRequestedAction';

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
        expect(screen.getByText('False positive')).toBeInTheDocument();
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
        expect(screen.getByText('Deferral (until fixed)')).toBeInTheDocument();
    });

    it('should show the requested action for a 2 week deferral', () => {
        const currentDate = new Date();
        const daysToAdd = 14;
        const expiresOnDate = addDays(currentDate, daysToAdd);

        render(
            <VulnRequestedAction
                targetState="DEFERRED"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: false,
                    expiresOn: expiresOnDate.toISOString(),
                }}
                currentDate={currentDate}
            />
        );
        expect(screen.getByText('Deferral (14 days)')).toBeInTheDocument();
    });

    it('should show the requested action for a 30 day deferral', () => {
        const currentDate = new Date();
        const daysToAdd = 30;
        const expiresOnDate = addDays(currentDate, daysToAdd);

        render(
            <VulnRequestedAction
                targetState="DEFERRED"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: false,
                    expiresOn: expiresOnDate.toISOString(),
                }}
                currentDate={currentDate}
            />
        );
        expect(screen.getByText('Deferral (30 days)')).toBeInTheDocument();
    });

    it('should show the requested action for a 90 day deferral', () => {
        const currentDate = new Date();
        const daysToAdd = 90;
        const expiresOnDate = addDays(currentDate, daysToAdd);

        render(
            <VulnRequestedAction
                targetState="DEFERRED"
                requestStatus="PENDING"
                deferralReq={{
                    expiresWhenFixed: false,
                    expiresOn: expiresOnDate.toISOString(),
                }}
                currentDate={currentDate}
            />
        );
        expect(screen.getByText('Deferral (90 days)')).toBeInTheDocument();
    });

    // @TODO: Add test for indefinite deferral
});
