import { Children, useState } from 'react';

import Tab from 'Components/Tab';

/**
 * This hook enables the behavior of selecting between different tabs and seeing a new view
 *
 * @param {ReactComponent[]} children - contains the child components to render. It should contain components with the type provided in the second param
 * @param {ReactComponent} Tab - the child tab component used
 *
 * Example: check out useTabs.test.js to see how to use the hook
 *
 */
function useTabs(children) {
    const [activeTabIndex, selectActiveTabIndex] = useState(0);

    const tabHeaders = Children.toArray(children).map((child, i) => {
        const {
            type,
            props: { title, dataTestId },
        } = child;

        if (type !== Tab) {
            throw Error(
                `The "useTabs" hook can only take children of type (${Tab.name}). A child of type (${type}) was provided.`
            );
        }
        if (!title) {
            throw Error(
                `The "useTabs" hook must include children of type (${Tab.name}) that have a (title) prop`
            );
        }

        const isActive = activeTabIndex === i;
        function onSelectTab() {
            selectActiveTabIndex(i);
        }

        return { title, isActive, onSelectTab, dataTestId };
    });

    const activeTabContent = Children.toArray(children)[activeTabIndex];

    return { tabHeaders, activeTabContent };
}

export default useTabs;
