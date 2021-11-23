/* eslint-disable jest/no-disabled-tests */
import React from 'react';
import { render, screen } from '@testing-library/react';

import RequestedAction from './RequestedAction';

describe('RequestedAction', () => {
    it('should show requested action for false positives', () => {
        const deferralReq = {};
        render(<RequestedAction targetState="FALSE_POSITIVE" deferralReq={deferralReq} />);
        expect(screen.getByText('Mark false positive')).toBeDefined();
    });

    it('should show the requested action for an until fixed deferral', () => {
        const deferralReq = {
            expiresWhenFixed: true,
        };
        render(<RequestedAction targetState="DEFERRED" deferralReq={deferralReq} />);
        expect(screen.getByText('Expire when fixed')).toBeDefined();
    });

    // @TODO: Figure out why sometimes it renders "14 days" and sometimes it renders one day less
    it.skip('should show the requested action for a 2 week deferral', () => {
        const currentDate = new Date();
        const daysToAdd = 14;
        currentDate.setDate(currentDate.getDate() + daysToAdd);

        const deferralReq = {
            expiresOn: currentDate.toISOString(),
        };

        render(<RequestedAction targetState="DEFERRED" deferralReq={deferralReq} />);
        expect(screen.getByText('14 days')).toBeDefined();
    });

    // @TODO: Figure out why sometimes it renders "1 month" and sometimes it renders one day less
    it.skip('should show the requested action for a 30 day deferral', () => {
        const currentDate = new Date();
        const daysToAdd = 30;
        currentDate.setDate(currentDate.getDate() + daysToAdd);

        const deferralReq = {
            expiresOn: currentDate.toISOString(),
        };

        render(<RequestedAction targetState="DEFERRED" deferralReq={deferralReq} />);
        expect(screen.getByText('1 month')).toBeDefined();
    });

    // @TODO: Figure out why sometimes it renders "3 month" and sometimes it renders one month less
    it.skip('should show the requested action for a 90 day deferral', () => {
        const currentDate = new Date();
        const daysToAdd = 90;
        currentDate.setDate(currentDate.getDate() + daysToAdd);

        const deferralReq = {
            expiresOn: currentDate.toISOString(),
        };

        render(<RequestedAction targetState="DEFERRED" deferralReq={deferralReq} />);
        expect(screen.getByText('3 months')).toBeDefined();
    });

    // @TODO: Add test for indefinite deferral
});
