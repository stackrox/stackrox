import React from 'react';

import {
    Button,
    Divider,
    Flex,
    Form,
    FormGroup,
    Grid,
    GridItem,
    PageSection,
    Split,
    SplitItem,
    Switch,
    Text,
    TextInput,
    Title,
} from '@patternfly/react-core';

type BaseSettingProps = {
    isEnabled: boolean;
};

function NumericSetting({ value, isEnabled }: BaseSettingProps & { value: number }) {
    return (
        <>
            <GridItem span={8} md={4} xl={3}>
                <FormGroup>
                    <Flex direction={{ default: 'row' }} flexWrap={{ default: 'nowrap' }}>
                        <TextInput type="number" style={{ width: '70px' }} value={value} />
                        <span>days</span>
                    </Flex>
                </FormGroup>
            </GridItem>
            <GridItem span={4} md={8} xl={9}>
                <FormGroup>
                    <Switch label="Enabled" labelOff="Disabled" isChecked={isEnabled} />
                </FormGroup>
            </GridItem>
        </>
    );
}

function BooleanSetting({ label, isEnabled }: BaseSettingProps & { label: string }) {
    return (
        <>
            <GridItem className="pf-u-py-xs" span={8} md={4} xl={3}>
                <p>{label}</p>
            </GridItem>
            <GridItem className="pf-u-py-xs" span={4} md={8} xl={9}>
                <FormGroup>
                    <Switch label="Enabled" labelOff="Disabled" isChecked={isEnabled} />
                </FormGroup>
            </GridItem>
        </>
    );
}

function VulnerabilitiesConfiguration() {
    return (
        <>
            <div className="pf-u-py-md pf-u-px-md pf-u-px-lg-on-xl">
                <Split className="pf-u-align-items-center">
                    <SplitItem isFilled>
                        <Text>Configure deferral behavior for vulnerabilities</Text>
                    </SplitItem>
                    <SplitItem>
                        <Button variant="primary">Save</Button>
                    </SplitItem>
                </Split>
            </div>
            <Divider component="div" />
            <PageSection variant="light" component="div">
                <Title headingLevel="h2">Configure deferral times</Title>
                <Form className="pf-u-py-lg">
                    <Grid hasGutter>
                        <NumericSetting value={14} isEnabled />
                        <NumericSetting value={30} isEnabled />
                        <NumericSetting value={60} isEnabled />
                        <NumericSetting value={90} isEnabled />
                        <BooleanSetting label="Indefinitely" isEnabled />
                        <BooleanSetting label="Expires when all CVEs fixable" isEnabled />
                        <BooleanSetting label="Expires when any CVE fixable" isEnabled />
                        <BooleanSetting label="Allow custom date" isEnabled />
                    </Grid>
                </Form>
            </PageSection>
        </>
    );
}

export default VulnerabilitiesConfiguration;
