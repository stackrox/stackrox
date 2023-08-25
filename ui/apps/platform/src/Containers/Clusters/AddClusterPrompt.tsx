import React from 'react';
import {
    Alert,
    AlertVariant,
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
    Text,
    TextContent,
    TextVariants,
} from '@patternfly/react-core';

function AddClusterPrompt() {
    return (
        <>
            <Alert isInline variant={AlertVariant.success} title="You are ready to go!">
                <p className="pf-u-font-weight-normal">
                    You have successfully deployed a Red Hat Advanced Cluster Security platform. Now
                    you can configure the clusters you want to secure.
                </p>
            </Alert>
            <Flex
                alignItems={{ default: 'alignItemsCenter' }}
                justifyContent={{ default: 'justifyContentCenter' }}
                className="pf-u-text-align-center"
                direction={{ default: 'column' }}
            >
                <FlexItem className="pf-u-w-66 pf-u-pt-xl">
                    <TextContent className="pf-u-mb-md">
                        <Text component={TextVariants.h2} className="pf-u-font-size-2xl">
                            Configure the clusters you want to secure.
                        </Text>
                        <Text component={TextVariants.p} className="pf-u-font-weight-normal">
                            Follow the instructions to add secured clusters for Central to monitor.
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
        </>
    );
}

export default AddClusterPrompt;
