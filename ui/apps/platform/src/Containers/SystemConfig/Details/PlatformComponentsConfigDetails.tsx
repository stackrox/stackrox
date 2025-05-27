import React, { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    CodeBlock,
    Divider,
    Grid,
    GridItem,
    Stack,
    Text,
} from '@patternfly/react-core';

import { PlatformComponentsConfig } from 'types/config.proto';

import './PlatformComponentsConfigDetails.css';
import RedHatLayeredProductsCard from './components/RedHatLayeredProductsCard';
import CustomPlatformComponentsCard from './components/CustomPlatformComponentsCard';
import { getPlatformComponentsConfigRules } from '../configUtils';

export type PlatformComponentsConfigDetailsProps = {
    platformComponentsConfig: PlatformComponentsConfig;
};

const PlatformComponentsConfigDetails = ({
    platformComponentsConfig,
}: PlatformComponentsConfigDetailsProps): ReactElement => {
    const { coreSystemRule, redHatLayeredProductsRule, customRules } =
        getPlatformComponentsConfigRules(platformComponentsConfig);

    return (
        <Grid hasGutter>
            <GridItem sm={12} md={6} lg={4}>
                <Card isFlat>
                    <CardTitle>Core system</CardTitle>
                    <CardBody>
                        <Stack hasGutter>
                            <Text>
                                Components found in core Openshift and Kubernetes namespaces are
                                included in the platform definition by default.
                            </Text>
                            <Divider component="div" />
                            <Text component="small" className="pf-v5-u-color-200">
                                Namespaces match (Regex)
                            </Text>
                            <CodeBlock>{coreSystemRule?.namespaceRule?.regex || 'None'}</CodeBlock>
                        </Stack>
                    </CardBody>
                </Card>
            </GridItem>
            <GridItem sm={12} md={6} lg={4}>
                <RedHatLayeredProductsCard rule={redHatLayeredProductsRule} />
            </GridItem>
            <GridItem sm={12} md={6} lg={4}>
                <CustomPlatformComponentsCard customRules={customRules} />
            </GridItem>
        </Grid>
    );
};

export default PlatformComponentsConfigDetails;
