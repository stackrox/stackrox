export type MetricSample = {
    metricName: string;
    labels: Record<string, string>;
    value: string;
    timestamp?: number;
};

export type ParsedMetrics = {
    metrics: MetricSample[];
    metricNames: string[];
};

export type MetricSelector = {
    id: string;
    metricName: string;
};
