import React, { useMemo } from 'react';
import {
    Tab,
    TabContent,
    TabContentProps,
    TabTitleText,
    Tabs,
    TabsComponent,
    TabsProps,
} from '@patternfly/react-core';
import noop from 'lodash/noop';

import useURLStringUnion from 'hooks/useURLStringUnion';

export type TabDescriptor = {
    key: string;
    content: React.ReactNode;
    contentProps?: Omit<TabContentProps, 'id' | 'ref' | 'eventKey'>;
};

export type UseTabContentOptions<TabTuple extends TabDescriptor[]> = {
    parameterName: string;
    tabKeys: Parameters<typeof useURLStringUnion>[1];
    tabs: [...TabTuple];
    tabsProps?: Omit<TabsProps, 'children' | 'ref'>;
    onTabChange?: TabsProps['onSelect'];
};

export default function useTabContent<TabTuple extends TabDescriptor[]>({
    parameterName,
    tabKeys,
    tabs,
    tabsProps = {},
    onTabChange = noop,
}: UseTabContentOptions<TabTuple>): [JSX.Element, (JSX.Element | null)[]] {
    const [activeKey, setActiveTabKey] = useURLStringUnion(parameterName, tabKeys);

    const refsById = useMemo(() => {
        const refs = {};
        tabs.forEach(({ key }) => {
            refs[key] = React.createRef<HTMLElement>();
        });
        return refs;
    }, [tabs]);

    const tabComponents = tabs.map(({ key, content, contentProps }) => {
        const id = `${key}-tab`;
        const ref = refsById[key];
        return [
            <Tab
                key={key}
                eventKey={key}
                title={<TabTitleText>{key}</TabTitleText>}
                tabContentId={id}
                tabContentRef={ref}
            />,
            activeKey === key ? (
                <TabContent {...contentProps} id={id} ref={ref} key={key} eventKey={key}>
                    {content}
                </TabContent>
            ) : null,
        ];
    });

    const tabsComponent = (
        <Tabs
            {...tabsProps}
            activeKey={activeKey}
            onSelect={(e, key) => {
                setActiveTabKey(key);
                onTabChange?.(e, key);
            }}
            component={TabsComponent.nav}
            role="region"
            mountOnEnter
            unmountOnExit
        >
            {tabComponents.map(([tab]) => tab)}
        </Tabs>
    );

    return [tabsComponent, tabComponents.map(([, content]) => content)];
}
