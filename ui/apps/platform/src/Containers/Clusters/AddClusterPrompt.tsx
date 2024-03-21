import React from 'react';
import {
    Button,
    ButtonVariant,
    EmptyState,
    EmptyStateIcon,
    Flex,
    FlexItem,
    Text,
    TextContent,
    TextVariants,
    EmptyStateHeader,
    EmptyStateFooter,
} from '@patternfly/react-core';
import { CheckCircleIcon } from '@patternfly/react-icons';

function AddClusterPrompt() {
    return (
        <EmptyState>
            <EmptyStateHeader
                icon={
                    <EmptyStateIcon
                        icon={CheckCircleIcon}
                        color="var(--pf-v5-global--success-color--100)"
                    />
                }
            />
            <EmptyStateFooter>
                <p className="pf-v5-u-font-weight-normal">
                    You have successfully deployed a Red Hat Advanced Cluster Security platform. Now
                    you can configure the clusters you want to secure.
                </p>
                <Flex
                    alignItems={{ default: 'alignItemsCenter' }}
                    justifyContent={{ default: 'justifyContentCenter' }}
                    className="pf-v5-u-text-align-center"
                    direction={{ default: 'column' }}
                >
                    <FlexItem className="pf-v5-u-w-66 pf-v5-u-pt-xl">
                        <TextContent className="pf-v5-u-mb-md">
                            <Text component={TextVariants.h2} className="pf-v5-u-font-size-2xl">
                                Configure the clusters you want to secure.
                            </Text>
                            <Text component={TextVariants.p} className="pf-v5-u-font-weight-normal">
                                Follow the instructions to add secured clusters for Central to
                                monitor.
                                <br />
                                Upon successful installation, secured clusters are listed here.
                            </Text>
                        </TextContent>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            variant={ButtonVariant.primary}
                            component="a"
                            target="_blank"
                            rel="noopener noreferrer nofollow"
                            href="https://docs.openshift.com/acs/installing/install-ocp-operator.html#adding-a-new-cluster-to-rhacs"
                        >
                            View instructions
                        </Button>
                    </FlexItem>
                </Flex>
            </EmptyStateFooter>
        </EmptyState>
    );
}

export default AddClusterPrompt;
