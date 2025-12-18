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

    for (let i = 0; i < labelsString.length; i += 1) {
        const char = labelsString[i];

        if (escapeNext) {
            currentPair += char;
            escapeNext = false;
        } else if (char === '\\') {
            currentPair += char;
            escapeNext = true;
        } else if (char === '"') {
            insideQuotes = !insideQuotes;
            currentPair += char;
        } else if (char === ',' && !insideQuotes) {
            if (currentPair.trim()) {
                pairs.push(currentPair);
            }
            currentPair = '';
        } else {
            currentPair += char;
        }
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
 * HELP comments: # HELP metric_name description
 */
export function parsePrometheusMetrics(text: string): ParsedMetrics {
    const metrics: MetricSample[] = [];
    const metricNamesSet = new Set<string>();
    const metricInfoMap: Record<string, { name: string; help?: string }> = {};

    const lines = text.split('\n');

    lines.forEach((line) => {
        const trimmedLine = line.trim();

        // Skip empty lines
        if (!trimmedLine) {
            return;
        }

        // Parse HELP comments
        if (trimmedLine.startsWith('# HELP ')) {
            const helpMatch = trimmedLine.match(/^# HELP\s+([a-zA-Z_:][a-zA-Z0-9_:.-]*)\s+(.*)$/);
            if (helpMatch) {
                const metricName = helpMatch[1];
                const helpText = helpMatch[2];
                // Add metric name even if there's no data yet
                metricNamesSet.add(metricName);
                if (!metricInfoMap[metricName]) {
                    metricInfoMap[metricName] = { name: metricName };
                }
                metricInfoMap[metricName].help = helpText;
            }
            return;
        }

        // Skip other comments
        if (trimmedLine.startsWith('#')) {
            return;
        }

        const parsed = parseMetricLine(trimmedLine);
        if (parsed) {
            metrics.push(parsed);
            metricNamesSet.add(parsed.metricName);
            // Ensure metric exists in info map even without HELP
            if (!metricInfoMap[parsed.metricName]) {
                metricInfoMap[parsed.metricName] = { name: parsed.metricName };
            }
        } else if (trimmedLine && !trimmedLine.startsWith('#')) {
            // eslint-disable-next-line no-console
            console.log('Failed to parse line:', trimmedLine.substring(0, 100));
        }
    });

    return {
        metrics,
        metricNames: Array.from(metricNamesSet).sort(),
        metricInfoMap,
    };
}

function parseMetricLine(line: string): MetricSample | null {
    // Match metric_name{labels} value [timestamp]
    // or metric_name value [timestamp]
    // Relaxed pattern to allow common non-standard metric names (with hyphens, dots, etc.)

    const metricNameMatch = line.match(/^([a-zA-Z_:][a-zA-Z0-9_:.-]*)/);
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
        labelPairs.forEach((pair) => {
            const trimmedPair = pair.trim();
            if (!trimmedPair) {
                return;
            }

            const equalIndex = trimmedPair.indexOf('=');
            if (equalIndex === -1) {
                return;
            }

            const labelName = trimmedPair.substring(0, equalIndex).trim();
            let labelValue = trimmedPair.substring(equalIndex + 1).trim();

            // Remove quotes
            if (labelValue.startsWith('"') && labelValue.endsWith('"')) {
                labelValue = labelValue.substring(1, labelValue.length - 1);
                // Unescape special characters
                labelValue = labelValue
                    .replace(/\\"/g, '"')
                    .replace(/\\\\/g, '\\')
                    .replace(/\\n/g, '\n');
            }

            labels[labelName] = labelValue;
        });
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
