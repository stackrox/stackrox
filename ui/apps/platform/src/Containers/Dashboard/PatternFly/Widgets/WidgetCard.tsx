import React, { ReactNode } from 'react';
import { Skeleton, Card, CardBody, CardHeader } from '@patternfly/react-core';

import { defaultChartHeight } from 'utils/chartUtils';
import WidgetErrorEmptyState from './WidgetErrorEmptyState';

type WidgetCardProps = {
    isLoading: boolean;
    error?: Error;
    errorTitle?: string;
    errorMessage?: string;
    header: ReactNode;
    children: ReactNode;
};

const height = `${defaultChartHeight}px` as const;

function WidgetCard({
    isLoading,
    error,
    errorTitle,
    errorMessage,
    header,
    children,
}: WidgetCardProps) {
    let cardContent: ReactNode;

    if (isLoading && !error) {
        cardContent = <Skeleton height={height} screenreaderText="Loading widget data" />;
    } else if (error) {
        cardContent = (
            <WidgetErrorEmptyState height={height} title={errorTitle || 'Unable to load data'}>
                {errorMessage || 'There was an error loading data for this widget'}
            </WidgetErrorEmptyState>
        );
    } else {
        cardContent = children;
    }

    return (
        <Card className="pf-u-h-100">
            <CardHeader>
                <div className="pf-u-flex-grow-1">{header}</div>
            </CardHeader>
            <CardBody>{cardContent}</CardBody>
        </Card>
    );
}

export default WidgetCard;
