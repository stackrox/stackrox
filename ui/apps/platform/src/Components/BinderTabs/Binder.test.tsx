import React from 'react';
import { render, screen } from '@testing-library/react';

import BinderTabs from './BinderTabs';
import Tab from '../Tab';

describe('BinderTabs', () => {
    test("renders the first tab's contents", () => {
        render(
            <BinderTabs>
                <Tab title="tab 1">Tab 1 Content</Tab>
                <Tab title="tab 2">Tab 2 Content</Tab>
            </BinderTabs>
        );
        expect(screen.getByText('Tab 1 Content')).toBeDefined();
    });

    test("selecting a new tab render's the new tab's contents", () => {
        render(
            <BinderTabs>
                <Tab title="tab 1">Tab 1 Content</Tab>
                <Tab title="tab 2">Tab 2 Content</Tab>
            </BinderTabs>
        );

        screen.getByText('tab 2').click();

        expect(screen.getByText('Tab 2 Content')).toBeDefined();
    });
});
