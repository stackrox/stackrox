import React, { useState } from 'react';
import {
    ExpandableSection,
    Badge,
    Flex,
    DataList,
    DataListCell,
    DataListItem,
    DataListItemCells,
    DataListItemRow,
} from '@patternfly/react-core';

export type ExpandableLabelSectionProps = {
    toggleText: string;
    labels: {
        key: string;
        value: string;
    }[];
};

function ExpandableLabelSection({ toggleText, labels }: ExpandableLabelSectionProps) {
    const [isExpanded, setIsExpanded] = useState(false);

    const onToggle = (_event: React.MouseEvent, isExpanded: boolean) => {
        setIsExpanded(isExpanded);
    };

    if (labels.length === 0) {
        return null;
    }

    return (
        <ExpandableSection
            toggleContent={
                <Flex
                    spaceItems={{ default: 'spaceItemsSm' }}
                    alignItems={{ default: 'alignItemsCenter' }}
                >
                    <span>{toggleText}</span>
                    <Badge isRead>{labels.length}</Badge>
                </Flex>
            }
            onToggle={onToggle}
            isExpanded={isExpanded}
        >
            <DataList aria-label={toggleText} isCompact>
                {labels.map(({ key, value }) => (
                    <DataListItem key={key}>
                        <DataListItemRow>
                            <DataListItemCells
                                dataListCells={[
                                    <DataListCell isFilled={false} key={key}>
                                        {key} : {value}
                                    </DataListCell>,
                                ]}
                            />
                        </DataListItemRow>
                    </DataListItem>
                ))}
            </DataList>
        </ExpandableSection>
    );
}

export default ExpandableLabelSection;
