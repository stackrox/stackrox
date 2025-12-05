import type { MetricSample, ParsedMetrics } from './types';

/**
 * Split label pairs by comma, respecting quoted strings.
 * Example: 'foo="bar",baz="qux,quux"' => ['foo="bar"', 'baz="qux,quux"']
 */
function splitLabelPairs(labelsString: string): string[] {
    const pairs: string[] = [];
    let currentPair = '';
    let insideQuotes = false;
    let escapeNext = false;

    for (let i = 0; i < labelsString.length; i++) {
        const char = labelsString[i];

        if (escapeNext) {
            currentPair += char;
            escapeNext = false;
            continue;
        }

        if (char === '\\') {
            currentPair += char;
            escapeNext = true;
            continue;
        }

        if (char === '"') {
            insideQuotes = !insideQuotes;
            currentPair += char;
            continue;
        }

        if (char === ',' && !insideQuotes) {
            if (currentPair.trim()) {
                pairs.push(currentPair);
            }
            currentPair = '';
            continue;
        }

        currentPair += char;
    }

    // Add the last pair
    if (currentPair.trim()) {
        pairs.push(currentPair);
    }

    return pairs;
}

/**
 * Parse Prometheus text exposition format
 * Format: metric_name{label1="value1",label2="value2"} value [timestamp]
 */
export function parsePrometheusMetrics(text: string): ParsedMetrics {
    const metrics: MetricSample[] = [];
    const metricNamesSet = new Set<string>();

    const lines = text.split('\n');

    for (const line of lines) {
        const trimmedLine = line.trim();

        // Skip comments and empty lines
        if (!trimmedLine || trimmedLine.startsWith('#')) {
            continue;
        }

        try {
            const parsed = parseMetricLine(trimmedLine);
            if (parsed) {
                metrics.push(parsed);
                metricNamesSet.add(parsed.metricName);
            }
        } catch (error) {
            // Skip malformed lines
            console.warn('Failed to parse metric line:', trimmedLine, error);
        }
    }

    return {
        metrics,
        metricNames: Array.from(metricNamesSet).sort(),
    };
}

function parseMetricLine(line: string): MetricSample | null {
    // Match metric_name{labels} value [timestamp]
    // or metric_name value [timestamp]

    const metricNameMatch = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)/);
    if (!metricNameMatch) {
        return null;
    }

    const metricName = metricNameMatch[1];
    let rest = line.substring(metricName.length);

    const labels: Record<string, string> = {};

    // Check if labels exist
    if (rest.trimStart().startsWith('{')) {
        const labelsEndIndex = rest.indexOf('}');
        if (labelsEndIndex === -1) {
            return null;
        }

        const labelsString = rest.substring(rest.indexOf('{') + 1, labelsEndIndex);
        rest = rest.substring(labelsEndIndex + 1);

        // Parse labels: label1="value1",label2="value2"
        // We need to split by commas but respect quoted strings
        const labelPairs = splitLabelPairs(labelsString);
        for (const pair of labelPairs) {
            const trimmedPair = pair.trim();
            if (!trimmedPair) {
                continue;
            }

            const equalIndex = trimmedPair.indexOf('=');
            if (equalIndex === -1) {
                continue;
            }

            const labelName = trimmedPair.substring(0, equalIndex).trim();
            let labelValue = trimmedPair.substring(equalIndex + 1).trim();

            // Remove quotes
            if (labelValue.startsWith('"') && labelValue.endsWith('"')) {
                labelValue = labelValue.substring(1, labelValue.length - 1);
                // Unescape special characters
                labelValue = labelValue.replace(/\\"/g, '"').replace(/\\\\/g, '\\').replace(/\\n/g, '\n');
            }

            labels[labelName] = labelValue;
        }
    }

    // Parse value and optional timestamp
    const parts = rest.trim().split(/\s+/);
    if (parts.length === 0) {
        return null;
    }

    const value = parts[0];
    const timestamp = parts.length > 1 ? parseInt(parts[1], 10) : undefined;

    return {
        metricName,
        labels,
        value,
        timestamp,
    };
}
