import React from 'react';
import { render, screen } from '@testing-library/react';

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

    test("selecting a new tab render's the new tab's contents", () => {
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

        screen.getByText('tab 2').click();

        expect(screen.getByText('Tab 2 Content')).toBeDefined();
    });
});
