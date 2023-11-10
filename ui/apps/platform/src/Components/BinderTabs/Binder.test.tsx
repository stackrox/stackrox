import React from 'react';
import { render, screen, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import BinderTabs from './BinderTabs';
import Tab from '../Tab';

describe('BinderTabs', () => {
    test("renders the first tab's contents", () => {
        render(
            <BinderTabs>
                <Tab title="tab 1">
                    <span>Tab 1 Content</span>
                </Tab>
                <Tab title="tab 2">
                    <span>Tab 2 Content</span>
                </Tab>
            </BinderTabs>
        );
        expect(screen.getByText('Tab 1 Content')).toBeDefined();
    });

    test("selecting a new tab render's the new tab's contents", async () => {
        const user = userEvent.setup();
        render(
            <BinderTabs>
                <Tab title="tab 1">
                    <span>Tab 1 Content</span>
                </Tab>
                <Tab title="tab 2">
                    <span>Tab 2 Content</span>
                </Tab>
            </BinderTabs>
        );

        await act(() => user.click(screen.getByText('tab 2')));

        expect(screen.getByText('Tab 2 Content')).toBeDefined();
    });
});
