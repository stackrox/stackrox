import { useState } from 'react';
import type { ReactElement } from 'react';
import {
    Card,
    CardBody,
    Flex,
    FormGroup,
    FormSection,
    Grid,
    GridItem,
    Tab,
    TabTitleText,
    Tabs,
    TextInput,
} from '@patternfly/react-core';
import type { FormikErrors, FormikValues } from 'formik';

import type { PrivateConfig, PrometheusMetricsCategory } from 'types/config.proto';

import {
    MetricCountBadge,
    PrometheusMetricsTable,
    categoryTitles,
    getMetricCount,
} from './PrometheusMetricsTabbedCard';

type PrometheusMetricsPeriodFormProps = {
    privateConfig: PrivateConfig;
    category: PrometheusMetricsCategory;
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>;
};

function PrometheusMetricsPeriodForm({
    privateConfig,
    category,
    onChange,
}: PrometheusMetricsPeriodFormProps): ReactElement {
    return (
        <FormGroup
            label="Gathering period in minutes (set to 0 to disable)"
            isRequired
            fieldId={`privateConfig.metrics.${category}.gatheringPeriodMinutes`}
        >
            <TextInput
                isRequired
                type="number"
                id={`privateConfig.metrics.${category}.gatheringPeriodMinutes`}
                name={`privateConfig.metrics.${category}.gatheringPeriodMinutes`}
                value={privateConfig?.metrics?.[category]?.gatheringPeriodMinutes}
                onChange={(event, value) => onChange(value, event)}
                min={0}
            />
        </FormGroup>
    );
}

export type PrometheusMetricsTabbedFormProps = {
    privateConfig: PrivateConfig;
    onChange: (value, event) => Promise<void> | Promise<FormikErrors<FormikValues>>;
    onCustomChange?: (
        value: unknown,
        id: string
    ) => Promise<void> | Promise<FormikErrors<FormikValues>>;
};

export default function PrometheusMetricsTabbedForm({
    privateConfig,
    onChange,
    onCustomChange,
}: PrometheusMetricsTabbedFormProps): ReactElement {
    const categories = Object.keys(categoryTitles) as PrometheusMetricsCategory[];
    const [activeTab, setActiveTab] = useState<PrometheusMetricsCategory>(categories[0]);

    return (
        <Card data-testid="prometheus-metrics-config">
            <Tabs
                activeKey={activeTab}
                onSelect={(_event, tabKey) => setActiveTab(tabKey as PrometheusMetricsCategory)}
            >
                {categories.map((category) => {
                    const metricCount = getMetricCount(
                        privateConfig?.metrics?.[category]?.descriptors
                    );

                    return (
                        <Tab
                            key={category}
                            eventKey={category}
                            title={
                                <TabTitleText>
                                    <Flex gap={{ default: 'gapSm' }}>
                                        {categoryTitles[category]}
                                        <MetricCountBadge count={metricCount} />
                                    </Flex>
                                </TabTitleText>
                            }
                        >
                            <CardBody>
                                <FormSection>
                                    <Grid hasGutter>
                                        <GridItem md={12}>
                                            <PrometheusMetricsPeriodForm
                                                privateConfig={privateConfig}
                                                category={category}
                                                onChange={onChange}
                                            />
                                        </GridItem>
                                        <GridItem md={12}>
                                            <FormGroup label="Metrics configuration" role="group">
                                                <PrometheusMetricsTable
                                                    descriptors={
                                                        privateConfig?.metrics?.[category]
                                                            ?.descriptors
                                                    }
                                                    category={category}
                                                    onCustomChange={onCustomChange}
                                                />
                                            </FormGroup>
                                        </GridItem>
                                    </Grid>
                                </FormSection>
                            </CardBody>
                        </Tab>
                    );
                })}
            </Tabs>
        </Card>
    );
}
