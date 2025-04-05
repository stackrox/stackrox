import React from 'react';
import { act, render, renderHook, screen } from '@testing-library/react';

import useTabs from './useTabs';
import Tab from '../Components/Tab';

const initialProps = {
    children: [
        <Tab key="tab 1" title="tab 1">
            Tab 1 Content
        </Tab>,
        <Tab key="tab 1" title="tab 2">
            Tab 2 Content
        </Tab>,
    ],
};

describe('useTabs', () => {
    it('should get the tab headers', () => {
        const { result } = renderHook(({ children }) => useTabs(children), {
            initialProps,
        });

        const { tabHeaders } = result.current;

        expect(tabHeaders[0].title).toEqual('tab 1');
        expect(tabHeaders[1].title).toEqual('tab 2');
    });

    it('should get the active tab content', async () => {
        const { result } = renderHook(({ children }) => useTabs(children), {
            initialProps,
        });

        const { tabHeaders, activeTabContent } = result.current;

        render(activeTabContent);

        expect(tabHeaders[0].isActive).toEqual(true);
        expect(tabHeaders[1].isActive).toEqual(false);
        expect(screen.getByText('Tab 1 Content')).toBeInTheDocument();
    });

    it("should select the second tab and see the second tab's content", async () => {
        const { result } = renderHook(({ children }) => useTabs(children), {
            initialProps,
        });

        // select the second tab header to change the active tab content
        act(() => {
            result.current.tabHeaders[1].onSelectTab();
        });

        const { tabHeaders, activeTabContent } = result.current;

        render(activeTabContent);

        expect(tabHeaders[0].isActive).toEqual(false);
        expect(tabHeaders[1].isActive).toEqual(true);
        expect(screen.getByText('Tab 2 Content')).toBeInTheDocument();
    });
});
