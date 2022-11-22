import React from 'react';
import {
    Button,
    Checkbox,
    Divider,
    Flex,
    FlexItem,
    Stack,
    StackItem,
    Switch,
    Tooltip,
} from '@patternfly/react-core';
import { HelpIcon } from '@patternfly/react-icons';

function DeploymentBaselines() {
    const [isAlertingOnViolations, setIsAlertingOnViolations] = React.useState<boolean>(false);
    const [isExcludingPortsAndProtocols, setIsExcludingPortsAndProtocols] =
        React.useState<boolean>(false);

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
                                onChange={setIsAlertingOnViolations}
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
                <Divider component="hr" />
                <StackItem isFilled>@TODO: Table</StackItem>
                <Divider component="hr" />
                <StackItem>
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsMd' }}
                        alignItems={{ default: 'alignItemsCenter' }}
                        justifyContent={{ default: 'justifyContentCenter' }}
                    >
                        <FlexItem>
                            <Checkbox
                                id="exclude-ports-and-protocols-checkbox"
                                label="Exclude ports & protocols"
                                isChecked={isExcludingPortsAndProtocols}
                                onChange={setIsExcludingPortsAndProtocols}
                            />
                        </FlexItem>
                        <FlexItem>
                            <Button variant="primary">Simulate baseline as network policy</Button>
                        </FlexItem>
                    </Flex>
                </StackItem>
            </Stack>
        </div>
    );
}

export default DeploymentBaselines;
