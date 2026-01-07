import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    CardTitle,
    CodeBlock,
    Content,
    Divider,
    Grid,
    GridItem,
    Stack,
} from '@patternfly/react-core';

import type { PlatformComponentsConfig } from 'types/config.proto';

import './PlatformComponentsConfigDetails.css';
import RedHatLayeredProductsCard from './components/RedHatLayeredProductsCard';
import CustomPlatformComponentsCard from './components/CustomPlatformComponentsCard';
import { getPlatformComponentsConfigRules } from '../configUtils';

export type PlatformComponentsConfigDetailsProps = {
    platformComponentConfig: PlatformComponentsConfig;
};

const PlatformComponentsConfigDetails = ({
    platformComponentConfig,
}: PlatformComponentsConfigDetailsProps): ReactElement => {
    const { coreSystemRule, redHatLayeredProductsRule, customRules } =
        getPlatformComponentsConfigRules(platformComponentConfig);

    return (
        <Grid hasGutter>
            <GridItem sm={12} md={6} lg={4}>
                <Card>
                    <CardTitle>Core system</CardTitle>
                    <CardBody>
                        <Stack hasGutter>
                            <Content component="p">
                                Components found in core Openshift and Kubernetes namespaces are
                                included in the platform definition by default.
                            </Content>
                            <Divider component="div" />
                            <Content component="small" className="pf-v6-u-color-200">
                                Namespaces match (Regex)
                            </Content>
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
