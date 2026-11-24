import { Badge, Tab, TabTitleIcon, TabTitleText, Tabs } from '@patternfly/react-core';

import SeverityIcons from 'Components/PatternFly/SeverityIcons';
import { severityTabValues } from '../../types';
import type { SeverityTab } from '../../types';
import { severityLabelToSeverity } from '../../utils/searchUtils';

export type SeverityTabsProps = {
    activeKey: SeverityTab;
    counts?: Partial<Record<SeverityTab, number>>;
    onChange: (severity: SeverityTab) => void;
};

function SeverityTabs({ activeKey, counts, onChange }: SeverityTabsProps) {
    return (
        <Tabs
            activeKey={activeKey}
            onSelect={(_event, key) => {
                if (severityTabValues.includes(key as SeverityTab) && key !== activeKey) {
                    onChange(key as SeverityTab);
                }
            }}
            isBox
        >
            {severityTabValues.map((severity) => {
                const severityEnum = severityLabelToSeverity(severity);
                const Icon = SeverityIcons[severityEnum];
                const count = counts?.[severity];

                return (
                    <Tab
                        key={severity}
                        eventKey={severity}
                        title={
                            <>
                                <TabTitleIcon>
                                    <Icon />
                                </TabTitleIcon>
                                <TabTitleText>{severity}</TabTitleText>
                                {count !== undefined && <Badge isRead>{count.toLocaleString()}</Badge>}
                            </>
                        }
                    />
                );
            })}
        </Tabs>
    );
}

export default SeverityTabs;
