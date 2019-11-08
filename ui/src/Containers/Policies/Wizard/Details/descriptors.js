import {
    enforcementActionLabels,
    lifecycleStageLabels,
    portExposureLabels,
    rbacPermissionLabels,
    envVarSrcLabels
} from 'messages/common';

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
    if (resource.memoryResourceRequest) {
        output.push(formatResourceValue('Memory request', resource.memoryResourceRequest, 'MB'));
    }
    if (resource.memoryResourceLimit) {
        output.push(formatResourceValue('Memory limit', resource.memoryResourceLimit, 'MB'));
    }
    if (resource.cpuResourceRequest) {
        output.push(formatResourceValue('CPU request', resource.cpuResourceRequest, 'Cores'));
    }
    if (resource.cpuResourceLimit) {
        output.push(formatResourceValue('CPU limit', resource.cpuResourceLimit, 'Cores'));
    }
    return output.join(', ');
};

const formatScope = (scope, props) => {
    if (!scope) return '';
    const values = [];
    if (scope.cluster !== '') {
        let { cluster } = scope;
        if (props.clustersById[scope.cluster]) {
            cluster = props.clustersById[scope.cluster].name;
        }
        values.push(`Cluster:${cluster}`);
    }
    if (scope.namespace !== '') {
        values.push(`Namespace:${scope.namespace}`);
    }
    if (scope.label) {
        values.push(`Label:${scope.label.key}=${scope.label.value}`);
    }
    return values.join('; ');
};

const formatDeploymentWhitelistScope = (whitelistScope, props) => {
    const values = [];
    if (whitelistScope.name !== '') {
        values.push(`Deployment Name:${whitelistScope.name}`);
    }
    const scopeVal = formatScope(whitelistScope.scope, props);
    if (scopeVal !== '') {
        values.push(scopeVal);
    }
    return values.join('; ');
};

// JSON value name mapped to formatting for description page.
const fieldsMap = {
    id: {
        label: 'ID',
        formatValue: d => d
    },
    name: {
        label: 'Name',
        formatValue: d => d
    },
    lifecycleStages: {
        label: 'Lifecycle Stage',
        formatValue: d => d.map(v => lifecycleStageLabels[v]).join(', ')
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
        label: 'Restricted to Scopes',
        formatValue: (d, props) =>
            d && d.length ? d.map(scope => formatScope(scope, props)) : null
    },
    enforcementActions: {
        label: 'Enforcement Action',
        formatValue: d => d.map(v => enforcementActionLabels[v]).join(', ')
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
        label: 'Whitelists',
        formatValue: (d, props) => {
            const whitelistObj = {};
            const deploymentWhitelistScopes = d
                .filter(obj => obj.deployment && (obj.deployment.name || obj.deployment.scope))
                .map(obj => obj.deployment);
            if (deploymentWhitelistScopes.length > 0) {
                whitelistObj['Deployment Whitelists'] = deploymentWhitelistScopes.map(
                    deploymentWhitelistScope =>
                        formatDeploymentWhitelistScope(deploymentWhitelistScope, props)
                );
            }
            const images = d
                .filter(obj => obj.image && obj.image.name !== '')
                .map(obj => obj.image.name);
            if (images.length !== 0) {
                whitelistObj['Image Whitelists'] = images;
            }
            return whitelistObj;
        }
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
        label: 'Days since image was created',
        formatValue: d => (d !== '0' ? `${Number(d)} Days ago` : '')
    },
    noScanExists: {
        label: 'Image Scan Status',
        formatValue: () => 'Verify that the image is scanned'
    },
    scanAgeDays: {
        label: 'Days since image was last scanned',
        formatValue: d => (d !== '0' ? `${Number(d)} Days ago` : '')
    },
    lineRule: {
        label: 'Dockerfile Line',
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
    fixedBy: {
        label: 'Fixed By',
        formatValue: d => d
    },
    component: {
        label: 'Image Component',
        formatValue: d => {
            const name = d.name ? `${d.name}` : '';
            const version = d.version ? d.version : '';
            return `"${name}" with version "${version}"`;
        }
    },
    env: {
        label: 'Environment Variable',
        formatValue: d => {
            const key = d.key ? `${d.key}` : '';
            const value = d.value ? d.value : '';
            const valueFrom = !d.envVarSource
                ? ''
                : ` Value From: ${envVarSrcLabels[d.envVarSource]}`;
            return `${key}=${value};${valueFrom}`;
        }
    },
    disallowedAnnotation: {
        label: 'Disallowed Annotation',
        formatValue: d => {
            const key = d.key ? `Key: ${d.key} ` : '';
            const value = d.value ? `Value: ${d.value}` : '';
            return `${key}${value}`;
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
            output.push(d.readOnly ? 'Writable: No' : 'Writable: Yes');
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
    readOnlyRootFs: {
        label: 'Read Only Root Filesystem',
        formatValue: d => (d === true ? 'Yes' : 'Not Enabled')
    },
    containerResourcePolicy: {
        label: 'Container Resources',
        formatValue: formatResources
    },
    processPolicy: {
        label: 'Process Execution',
        formatValue: d => {
            const name = d.name ? `Process matches name "${d.name}"` : 'Process';
            const args = d.args ? `and matches args "${d.args}"` : '';
            const ancestor = d.ancestor ? `and has ancestor matching "${d.ancestor}"` : '';
            const uid = d.uid ? `with uid ${d.uid}` : ``;
            return `${name} ${args} ${ancestor} ${uid}`;
        }
    },
    portExposurePolicy: {
        label: 'Port Exposure',
        formatValue: d => {
            const output = d.exposureLevels.map(element => portExposureLabels[element]);
            return output.join(', ');
        }
    },
    hostMountPolicy: {
        label: 'Host Mount Policy',
        formatValue: d => (d.readOnly ? 'Not Enabled' : 'Writable: Yes')
    },
    whitelistEnabled: {
        label: 'Whitelists Enabled',
        formatValue: d => (d ? 'Yes' : 'No')
    },
    permissionPolicy: {
        label: 'Minimum RBAC Permissions',
        formatValue: d => rbacPermissionLabels[d.permissionLevel]
    }
};

export default fieldsMap;
