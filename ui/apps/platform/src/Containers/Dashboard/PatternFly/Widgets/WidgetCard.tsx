import React, { ReactNode } from 'react';
import { Skeleton, Card, Title } from '@patternfly/react-core';

import { defaultChartHeight } from 'utils/chartUtils';
import WidgetErrorEmptyState from './WidgetErrorEmptyState';

type WidgetCardProps = {
    title: string;
    isLoading: boolean;
    error: Error | null;
    children: ReactNode;
};

const height = `${defaultChartHeight}px` as const;

function WidgetCard({ title, isLoading, error, children }: WidgetCardProps) {
    let cardContent: ReactNode;

    if (error) {
        cardContent = (
            <WidgetErrorEmptyState height={height} title="Unable to load data">
                There was an error loading data for this widget
            </WidgetErrorEmptyState>
        );
    } else if (isLoading) {
        cardContent = <Skeleton height={height} screenreaderText={`Loading ${title}`} />;
    } else {
        cardContent = children;
    }

    return (
        <Card>
            <Title headingLevel="h2" className="pf-u-p-md">
                {title}
            </Title>
            {cardContent}
        </Card>
    );
}

export default WidgetCard;
