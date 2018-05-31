const cvssMap = {
    mathOp: {
        MAX: 'Max score',
        AVG: 'Avg score',
        MIN: 'Min score'
    },
    op: {
        GREATER_THAN: 'Is greater than',
        GREATER_THAN_OR_EQUALS: 'Is greater than or equal to',
        EQUALS: 'Is equal to',
        LESS_THAN_OR_EQUALS: 'Is less than or equal to',
        LESS_THAN: 'Is less than'
    }
};

const fieldsMap = {
    id: {
        label: 'Id',
        formatValue: d => d
    },
    name: {
        label: 'Name',
        formatValue: d => d
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
        formatValue: d => d.map(o => o.cluster).join(', ')
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
                .filter(obj => obj.deployment.name !== undefined)
                .map(obj => obj.deployment.name)
                .join(', ')
    },
    imageName: {
        label: 'Image',
        formatValue: d => {
            const namespace = d.namespace ? d.namespace : 'any';
            const repo = d.repo ? d.repo : 'any';
            const tag = d.tag ? d.tag : 'any';
            const registry = d.registry ? d.registry : 'any';
            return `Alert on ${namespace} namespace${d.namespace ? '' : 's'} using ${repo} repo${
                d.repo ? '' : 's'
            } using ${tag} tag from ${registry} registry`;
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
        formatValue: d => `${cvssMap.mathOp[d.mathOp]} ${cvssMap.op[d.op]} ${d.value}`
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
            const type = d.type ? `${d.type} ` : '';
            const path = d.path ? d.path : '';
            return `${type}${path}`;
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
        formatValue: d => d
    },
    addCapabilities: {
        label: 'Add Capabilities',
        formatValue: d => d
    },
    privileged: {
        label: 'Privileged',
        formatValue: d => (d === true ? 'Yes' : 'No')
    }
};

export default fieldsMap;
