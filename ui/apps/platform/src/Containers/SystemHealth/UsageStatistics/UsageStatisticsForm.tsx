/* eslint-disable @typescript-eslint/no-unsafe-return */
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
    Grid,
    GridItem,
    yyyyMMddFormat,
} from '@patternfly/react-core';
import { MaxSecuredUnitsUsageResponse, SecuredUnitsUsage } from '../../../types/productUsage.proto';
import {
    downloadProductUsageCsv,
    fetchCurrentProductUsage,
    fetchMaxCurrentUsage,
} from '../../../services/ProductUsageService';
import { getAxiosErrorMessage } from '../../../utils/responseErrorUtils';

function UsageStatisticsForm(): ReactElement {
    const initialStartDate = new Date();
    initialStartDate.setDate(initialStartDate.getDate() - 30);
    const [startDate, setStartDate] = useState(initialStartDate);
    const [endDate, setEndDate] = useState(new Date());
    const [currentUsage, setCurrentUsage] = useState({
        numNodes: 0,
        numCpuUnits: 0,
    } as SecuredUnitsUsage);
    const [maxUsage, setMaxUsage] = useState({
        maxNodes: 0,
        maxNodesAt: '-',
        maxCpuUnits: 0,
        maxCpuUnitsAt: '-',
    } as MaxSecuredUnitsUsageResponse);
    const [errorFetchingCurrent, setErrorFetchingCurrent] = useState<string>('');
    const [errorFetchingMax, setErrorFetchingMax] = useState<string>('');

    useEffect(() => {
        fetchCurrentProductUsage()
            .then((usage) => {
                setCurrentUsage(usage.data);
                setErrorFetchingCurrent('');
            })
            .catch((error) => {
                setErrorFetchingCurrent(getAxiosErrorMessage(error));
            });
    }, []);
    useEffect(() => {
        fetchMaxCurrentUsage({ from: startDate.toISOString(), to: endDate.toISOString() })
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
            <div className="pf-u-font-size-lg">Currently secured</div>
            <Divider style={{ padding: '5px 0px 10px 0px' }} />
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
                    <DescriptionListTerm>Nodes count</DescriptionListTerm>
                    <DescriptionListDescription>{currentUsage.numNodes}</DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>

            <div className="pf-u-font-size-lg" style={{ padding: '20px 0px 0px 0px' }}>
                Maximum secured
            </div>
            <Divider style={{ padding: '5px 0px 10px 0px' }} />
            <DescriptionList
                columnModifier={{
                    default: '2Col',
                }}
            >
                <DescriptionListGroup>
                    <DescriptionListTerm>Start date</DescriptionListTerm>
                    <DescriptionListDescription>
                        <DatePicker
                            value={yyyyMMddFormat(startDate)}
                            onChange={(_str, _, date) => {
                                if (date) {
                                    setStartDate(date);
                                }
                            }}
                        />
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>End date</DescriptionListTerm>
                    <DescriptionListDescription>
                        <DatePicker
                            value={yyyyMMddFormat(endDate)}
                            onChange={(_str, _, date) => {
                                if (date) {
                                    // Add 23 hours 59 minutes to include end date completely in the request.
                                    date.setTime(date.getTime() + (24 * 60 - 1) * 60 * 1000);
                                    setEndDate(date);
                                }
                            }}
                        />
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>CPU units</DescriptionListTerm>
                    <DescriptionListDescription>{maxUsage.maxCpuUnits}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Nodes count</DescriptionListTerm>
                    <DescriptionListDescription>{maxUsage.maxNodes}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>CPU units collection date</DescriptionListTerm>
                    <DescriptionListDescription>
                        {maxUsage && maxUsage?.maxCpuUnitsAt
                            ? yyyyMMddFormat(new Date(maxUsage.maxCpuUnitsAt))
                            : '-'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                    <DescriptionListTerm>Nodes count collection date</DescriptionListTerm>
                    <DescriptionListDescription>
                        {maxUsage && maxUsage?.maxNodesAt
                            ? yyyyMMddFormat(new Date(maxUsage.maxNodesAt))
                            : '-'}
                    </DescriptionListDescription>
                </DescriptionListGroup>
            </DescriptionList>
            <Grid hasGutter style={{ padding: '20px 0px 0px 0px' }}>
                <GridItem span={12}>
                    <Button
                        onClick={() =>
                            downloadProductUsageCsv({
                                from: startDate.toISOString(),
                                to: endDate.toISOString(),
                            })
                        }
                    >
                        Download CSV
                    </Button>
                </GridItem>
            </Grid>
        </>
    );
}

export default UsageStatisticsForm;
