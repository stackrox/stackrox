import { useState } from 'react';

function useTabs({ defaultTab }: { defaultTab: string }) {
    const [activeKeyTab, setActiveKeyTab] = useState(defaultTab);

    function onSelectTab(event, eventKey) {
        event.preventDefault(); // without this, the page refreshes with empty query string :(
        setActiveKeyTab(eventKey);
    }

    return { activeKeyTab, onSelectTab };
}

export default useTabs;
