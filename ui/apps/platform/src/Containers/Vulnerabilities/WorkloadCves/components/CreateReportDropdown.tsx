import { useState } from 'react';
import type { MouseEvent as ReactMouseEvent, Ref } from 'react';
import { Dropdown, DropdownItem, DropdownList, MenuToggle } from '@patternfly/react-core';
import type { MenuToggleElement } from '@patternfly/react-core';

import useFeatureFlags from 'hooks/useFeatureFlags';
import { ensureExhaustive } from 'utils/type.utils';

const dropdownItems = [
    {
        text: 'Export report as CSV',
        description:
            'Export a view-based CSV report from this view using the filters you’ve applied.',
    },
    {
        text: 'Create scheduled report',
        description: 'Create a scheduled report from this view using the filters you’ve applied.',
        featureFlagDependency: 'ROX_VULNERABILITY_REPORTS_ENHANCED_FILTERING',
    },
] as const;

type DropdownItemText = (typeof dropdownItems)[number]['text'];

export type CreateReportDropdownProps = {
    onSelectExportReportAsCSV: () => void;
    onSelectCreateScheduledReport: () => void;
};

function CreateReportDropdown({
    onSelectExportReportAsCSV,
    onSelectCreateScheduledReport,
}: CreateReportDropdownProps) {
    const [isOpen, setIsOpen] = useState(false);
    const { isFeatureFlagEnabled } = useFeatureFlags();

    const onToggleClick = () => {
        setIsOpen((prev) => !prev);
    };

    const onSelectHandler = (
        _event: ReactMouseEvent<Element, MouseEvent> | undefined,
        value: DropdownItemText
    ) => {
        switch (value) {
            case 'Export report as CSV':
                onSelectExportReportAsCSV();
                break;
            case 'Create scheduled report':
                onSelectCreateScheduledReport();
                break;
            default:
                ensureExhaustive(value);
        }
        setIsOpen(false);
    };

    return (
        <>
            <Dropdown
                isOpen={isOpen}
                onSelect={onSelectHandler}
                onOpenChange={(isOpen: boolean) => setIsOpen(isOpen)}
                toggle={(toggleRef: Ref<MenuToggleElement>) => (
                    <MenuToggle ref={toggleRef} onClick={onToggleClick} isExpanded={isOpen}>
                        Create report
                    </MenuToggle>
                )}
                shouldFocusToggleOnSelect
                popperProps={{ position: 'right', appendTo: () => document.body }}
            >
                <DropdownList>
                    {dropdownItems.map((dropdownItem) => {
                        const { description, text } = dropdownItem;
                        const isEnabled =
                            !('featureFlagDependency' in dropdownItem) ||
                            isFeatureFlagEnabled(dropdownItem.featureFlagDependency); // DELETE NOT

                        return isEnabled ? (
                            <DropdownItem key={text} value={text} description={description}>
                                {text}
                            </DropdownItem>
                        ) : null;
                    })}
                </DropdownList>
            </Dropdown>
        </>
    );
}

export default CreateReportDropdown;
