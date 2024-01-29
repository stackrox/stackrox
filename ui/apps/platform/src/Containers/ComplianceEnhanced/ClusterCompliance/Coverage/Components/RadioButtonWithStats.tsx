import React, { useMemo } from 'react';
import {
    Card,
    CardTitle,
    CardBody,
    Radio,
    FlexItem,
    Flex,
    Label,
    LabelGroup,
} from '@patternfly/react-core';
import { BarsIcon, CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';

import { ComplianceScanStatsShim } from 'services/ComplianceEnhancedService';

import { getStatusCounts } from '../compliance.coverage.utils';

export type RadioButtonWithStatsProps = {
    scanStats: ComplianceScanStatsShim;
    isSelected: boolean;
    onSelected: (scanName: string) => void;
};

function RadioButtonWithStats({ scanStats, isSelected, onSelected }: RadioButtonWithStatsProps) {
    const { scanName } = scanStats;
    const { passCount, failCount, otherCount } = useMemo(
        () => getStatusCounts(scanStats.checkStats),
        [scanStats]
    );

    const onKeyDown = (event: React.KeyboardEvent) => {
        if ([' ', 'Enter'].includes(event.key)) {
            event.preventDefault();
            onSelected(scanName);
        }
    };

    return (
        <>
            <Card
                id={`selectable-card-${scanName}`}
                onKeyDown={onKeyDown}
                onClick={() => onSelected(scanName)}
                hasSelectableInput
                onSelectableInputChange={() => onSelected(scanName)}
                isSelectableRaised
                isSelected={isSelected}
                isCompact
                selectableInputAriaLabel={`results for scan: ${scanName}`}
            >
                <CardTitle className="pf-u-p-sm pf-u-pb-0">
                    <Flex justifyContent={{ default: 'justifyContentFlexEnd' }}>
                        <FlexItem>
                            <Radio
                                id={`selectable-card-radio-${scanName}`}
                                aria-label={`radio button for ${scanName} coverage`}
                                name={`selectable-card-radio-${scanName}`}
                                isChecked={isSelected}
                            />
                        </FlexItem>
                    </Flex>
                </CardTitle>
                <CardTitle>{scanName}</CardTitle>
                <CardBody>
                    <LabelGroup aria-label={`check results for ${scanName}`}>
                        <Label
                            aria-label={`number of passing checks: ${passCount}`}
                            className="pf-u-mr-xs"
                            icon={<CheckCircleIcon />}
                            color="green"
                        >
                            {passCount}
                        </Label>
                        <Label
                            aria-label={`number of failing checks: ${failCount}`}
                            className="pf-u-mr-xs"
                            icon={<ExclamationCircleIcon />}
                            color="red"
                        >
                            {failCount}
                        </Label>
                        <Label
                            aria-label={`number of other checks: ${otherCount}`}
                            icon={<BarsIcon />}
                            color="grey"
                        >
                            {otherCount}
                        </Label>
                    </LabelGroup>
                </CardBody>
            </Card>
        </>
    );
}

export default RadioButtonWithStats;
