import { lifecycleStageLabels } from 'messages/common';

const comparatorOp = {
    GREATER_THAN: '>',
    GREATER_THAN_OR_EQUALS: '>=',
    EQUALS: '=',
    LESS_THAN_OR_EQUALS: '<=',
    LESS_THAN: '<'
};

const formatResourceValue = (prefix, value, suffix) =>
    `${prefix} ${comparatorOp[value.op]} ${value.value} ${suffix}`;

const formatResources = resource => {
    const output = [];
    if (resource.memoryResourceRequest !== null) {
        output.push(formatResourceValue('Memory request', resource.memoryResourceRequest, 'MB'));
    }
    if (resource.memoryResourceLimit !== null) {
        output.push(formatResourceValue('Memory limit', resource.memoryResourceLimit, 'MB'));
    }
    if (resource.cpuResourceRequest !== null) {
        output.push(formatResourceValue('CPU request', resource.cpuResourceRequest, 'Cores'));
    }
    if (resource.cpuResourceLimit !== null) {
        output.push(formatResourceValue('CPU limit', resource.cpuResourceLimit, 'Cores'));
    }
    return output.join(', ');
};

// JSON value name mapped to formatting for description page.
const fieldsMap = {
    id: {
        label: 'Id',
        formatValue: d => d
    },
    name: {
        label: 'Name',
        formatValue: d => d
    },
    lifecycleStage: {
        label: 'Lifecycle Stage',
        formatValue: d => lifecycleStageLabels[d]
    },
    severity: {
        label: 'Severity',
        formatValue: d => {
            switch (d) {
                case 'CRITICAL_SEVERITY':
                    return 'Critical';
                case 'HIGH_SEVERITY':
                    return 'High';
                case 'MEDIUM_SEVERITY':
                    return 'Medium';
                case 'LOW_SEVERITY':
                    return 'Low';
                default:
                    return '';
            }
        }
    },
    description: {
        label: 'Description',
        formatValue: d => d
    },
    rationale: {
        label: 'Rationale',
        formatValue: r => r
    },
    remediation: {
        label: 'Remediation',
        formatValue: r => r
    },
    notifiers: {
        label: 'Notifications',
        formatValue: (d, props) =>
            props.notifiers
                .filter(n => d.includes(n.id))
                .map(n => n.name)
                .join(', ')
    },
    scope: {
        label: 'Restricted to Clusters',
        formatValue: (d, props) =>
            d
                .map(o => {
                    if (props.clustersById[o.cluster]) {
                        return props.clustersById[o.cluster].name;
                    }
                    // Fall back to returning the cluster id, if we can't find
                    // the mapping to the cluster name.
                    return o.cluster;
                })
                .join(', ')
    },
    enforcement: {
        label: 'Enforcement Action',
        formatValue: d => {
            switch (d) {
                case 'UNSET_ENFORCEMENT':
                    return 'None';
                case 'SCALE_TO_ZERO_ENFORCEMENT':
                    return 'Scale to Zero Replicas';
                case 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT':
                    return 'Add an Unsatisfiable Node Constraint';
                default:
                    return d;
            }
        }
    },
    disabled: {
        label: 'Enabled',
        formatValue: d => (d !== true ? 'Yes' : 'No')
    },
    categories: {
        label: 'Categories',
        formatValue: d => d.join(', ')
    },
    whitelists: {
        label: 'Whitelisted Deployments',
        formatValue: d =>
            d
                .filter(obj => obj.deployment.name !== undefined && obj.deployment.name !== '')
                .map(obj => obj.deployment.name)
                .join(', ')
    },
    imageName: {
        label: 'Image',
        formatValue: d => {
            const remote = d.remote ? `images named ${d.remote}` : 'any image';
            const tag = d.tag ? `tag ${d.tag}` : 'any tag';
            const registry = d.registry ? `registry ${d.registry}` : 'any registry';
            return `Alert on ${remote} using ${tag} from ${registry}`;
        }
    },
    imageAgeDays: {
        label: 'Image Created',
        formatValue: d => (d !== '0' ? `${Number(d)} Days ago` : '')
    },
    scanExists: {
        label: 'Scan Does Not Exist',
        formatValue: () => 'Verify that the image is scanned'
    },
    scanAgeDays: {
        label: 'Image Last Scanned',
        formatValue: d => (d !== '0' ? `${Number(d)} Days ago` : '')
    },
    lineRule: {
        label: 'Line Rule',
        formatValue: d => `${d.instruction} ${d.value}`
    },
    cvss: {
        label: 'CVSS',
        formatValue: d => `${comparatorOp[d.op]} ${d.value}`
    },
    cve: {
        label: 'CVE',
        formatValue: d => d
    },
    component: {
        label: 'Component',
        formatValue: d => {
            const name = d.name ? `${d.name}` : '';
            const version = d.version ? d.version : '';
            return `'${name}' with version '${version}'`;
        }
    },
    env: {
        label: 'Environment',
        formatValue: d => {
            const key = d.key ? `${d.key}` : '';
            const value = d.value ? d.value : '';
            return `${key}=${value}`;
        }
    },
    requiredLabel: {
        label: 'Required Label',
        formatValue: d => {
            const key = d.key ? `${d.key}` : '';
            const value = d.value ? d.value : '';
            return `${key}=${value}`;
        }
    },
    requiredAnnotation: {
        label: 'Required Annotation',
        formatValue: d => {
            const key = d.key ? `${d.key}` : '';
            const value = d.value ? d.value : '';
            return `${key}=${value}`;
        }
    },
    command: {
        label: 'Command',
        formatValue: d => d
    },
    arguments: {
        label: 'Arguments',
        formatValue: d => d
    },
    directory: {
        label: 'Directory',
        formatValue: d => d
    },
    user: {
        label: 'User',
        formatValue: d => d
    },
    volumePolicy: {
        label: 'Volume Policy',
        formatValue: d => {
            const output = [];
            if (d.name) {
                output.push(`Name: ${d.name}`);
            }
            if (d.type) {
                output.push(`Type: ${d.type}`);
            }
            if (d.source) {
                output.push(`Source: ${d.source}`);
            }
            if (d.destination) {
                output.push(`Dest: ${d.destination}`);
            }
            return output.join(', ');
        }
    },
    portPolicy: {
        label: 'Port',
        formatValue: d => {
            const protocol = d.protocol ? `${d.protocol} ` : '';
            const port = d.port ? d.port : '';
            return `${protocol}${port}`;
        }
    },
    dropCapabilities: {
        label: 'Drop Capabilities',
        formatValue: d => d.join(', ')
    },
    addCapabilities: {
        label: 'Add Capabilities',
        formatValue: d => d.join(', ')
    },
    privileged: {
        label: 'Privileged',
        formatValue: d => (d === true ? 'Yes' : 'No')
    },
    containerResourcePolicy: {
        label: 'Container Resources',
        formatValue: formatResources
    },
    processPolicy: {
        label: 'Process Name',
        formatValue: d => {
            const name = d.name ? `Process named "${d.name}"` : 'Process';
            const args = d.args ? `with args "${d.args}"` : '';
            return `${name} ${args}`;
        }
    }
};

export default fieldsMap;
