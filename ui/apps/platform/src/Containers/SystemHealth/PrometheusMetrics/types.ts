export type MetricSample = {
    labels: Record<string, string>;
    value: string;
    timestamp?: number;
};

export type ParseError = {
    line: string;
    lineNumber: number;
};

export type ParsedMetrics = {
    metrics: Record<string, MetricSample[]>;
    metricInfoMap: Record<string, string | undefined>;
    parseErrors: ParseError[];
};
