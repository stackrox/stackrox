import React, { ReactElement, useEffect, useState } from 'react';
import {
    Alert,
    Button,
    DatePicker,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Divider,
    Flex,
    FlexItem,
    Grid,
    GridItem,
    Title,
    yyyyMMddFormat,
} from '@patternfly/react-core';
import { MaxSecuredUnitsUsageResponse, SecuredUnitsUsage } from 'types/administrationUsage.proto';
import {
    downloadAdministrationUsageCsv,
    fetchCurrentAdministrationUsage,
    fetchMaxCurrentUsage,
} from 'services/AdministrationUsageService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import dateFns from 'date-fns';

function AdministrationUsageForm(): ReactElement {
    const initialStartDate = dateFns.subDays(new Date(), 30);
    const [startDate, setStartDate] = useState(initialStartDate);
    const [endDate, setEndDate] = useState(new Date());
    const [currentUsage, setCurrentUsage] = useState<SecuredUnitsUsage>({
        numNodes: 0,
        numCpuUnits: 0,
    });
    const [maxUsage, setMaxUsage] = useState<MaxSecuredUnitsUsageResponse>({
        maxNodes: 0,
        maxNodesAt: '-',
        maxCpuUnits: 0,
        maxCpuUnitsAt: '-',
    });
    const [errorFetchingCurrent, setErrorFetchingCurrent] = useState<string>('');
    const [errorFetchingMax, setErrorFetchingMax] = useState<string>('');

    useEffect(() => {
        fetchCurrentAdministrationUsage()
            .then((usage) => {
                setCurrentUsage(usage.data);
                setErrorFetchingCurrent('');
            })
            .catch((error) => {
                setErrorFetchingCurrent(getAxiosErrorMessage(error));
            });
    }, []);
    useEffect(() => {
        // Add 1 day to include end date completely in the request.
        const requestedEndDate = dateFns.addDays(endDate, 1);
        fetchMaxCurrentUsage({ from: startDate.toISOString(), to: requestedEndDate.toISOString() })
            .then((usage) => {
                setMaxUsage(usage);
                setErrorFetchingMax('');
            })
            .catch((error) => {
                setErrorFetchingMax(getAxiosErrorMessage(error));
            });
    }, [startDate, endDate]);

    return (
        <>
            {errorFetchingCurrent && (
                <Alert isInline title={errorFetchingCurrent} variant="danger" />
            )}
            {errorFetchingMax && <Alert isInline title={errorFetchingMax} variant="danger" />}
            <Title headingLevel="h2">Currently secured</Title>
            <Divider className="pf-u-pt-xs pf-u-pb-sm" />
            <DescriptionList
                columnModifier={{
                    default: '2Col',
                }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>CPU units</DescriptionListTerm>
                    <DescriptionListDescription>
                        {currentUsage.numCpuUnits}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Node count</DescriptionListTerm>
                    <DescriptionListDescription>{currentUsage.numNodes}</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>

            <Title headingLevel="h2" className="pf-u-pt-sm">
                Maximum secured
            </Title>
            <Divider className="pf-u-pt-xs pf-u-pb-sm" />
            <Flex className="pf-u-pb-xl">
                <FlexItem>
                    <Title headingLevel="h4">Start date</Title>
                    <DatePicker
                        value={yyyyMMddFormat(startDate)}
                        onChange={(_str, _, date) => {
                            if (date) {
                                setStartDate(date);
                            }
                        }}
                    />
                </FlexItem>
                <FlexItem>
                    <Title headingLevel="h4">End date</Title>
                    <DatePicker
                        value={yyyyMMddFormat(endDate)}
                        onChange={(_str, _, date) => {
                            if (date) {
                                setEndDate(date);
                            }
                        }}
                    />
                </FlexItem>
            </Flex>
            <DescriptionList
                columnModifier={{
                    default: '2Col',
                }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>CPU units</DescriptionListTerm>
                    <DescriptionListDescription>{maxUsage.maxCpuUnits}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Node count</DescriptionListTerm>
                    <DescriptionListDescription>{maxUsage.maxNodes}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>CPU units observation date</DescriptionListTerm>
                    <DescriptionListDescription>
                        {maxUsage?.maxCpuUnitsAt
                            ? yyyyMMddFormat(new Date(maxUsage.maxCpuUnitsAt))
                            : '-'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Node count observation date</DescriptionListTerm>
                    <DescriptionListDescription>
                        {maxUsage?.maxNodesAt ? yyyyMMddFormat(new Date(maxUsage.maxNodesAt)) : '-'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            <Grid hasGutter className="pf-u-pt-md">
                <GridItem span={12}>
                    <Button
                        onClick={() => {
                            // Add 1 day to include end date completely in the request.
                            const requestedEndDate = dateFns.addDays(endDate, 1);
                            return downloadAdministrationUsageCsv({
                                from: startDate.toISOString(),
                                to: requestedEndDate.toISOString(),
                            });
                        }}
                    >
                        Download CSV
                    </Button>
                </GridItem>
            </Grid>
        </>
    );
}

export default AdministrationUsageForm;
