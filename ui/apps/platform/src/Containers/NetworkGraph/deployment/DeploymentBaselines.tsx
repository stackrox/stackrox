import React from 'react';
import { Flex, FlexItem, Stack, StackItem, Switch, Tooltip } from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

function DeploymentBaselines() {
    const [isAlertingOnViolations, setIsAlertingOnViolations] = React.useState<boolean>(false);

    const handleAlertingOnViolations = (checked: boolean) => {
        setIsAlertingOnViolations(checked);
    };

    return (
        <div className="pf-u-h-100 pf-u-p-md">
            <Stack hasGutter>
                <StackItem>
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <FlexItem>
                            <Switch
                                id="simple-switch"
                                label="Alert on baseline violation"
                                isChecked={isAlertingOnViolations}
                                onChange={handleAlertingOnViolations}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Tooltip
                                content={
                                    <div>
                                        Trigger violations for network policies not in the baseline
                                    </div>
                                }
                            >
                                <HelpIcon className="pf-u-color-200" />
                            </Tooltip>
                        </FlexItem>
                    </Flex>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentBaselines;
