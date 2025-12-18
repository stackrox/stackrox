export type MetricSample = {
    metricName: string;
    labels: Record<string, string>;
    value: string;
    timestamp?: number;
};

export type MetricInfo = {
    name: string;
    help?: string;
};

export type ParsedMetrics = {
    metrics: MetricSample[];
    metricNames: string[];
    metricInfoMap: Record<string, MetricInfo>;
};

export type MetricSelector = {
    id: string;
    metricName: string;
};
