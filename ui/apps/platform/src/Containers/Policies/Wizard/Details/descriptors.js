import {
    enforcementActionLabels,
    lifecycleStageLabels,
    portExposureLabels,
    rbacPermissionLabels,
    envVarSrcLabels,
    seccompProfileTypeLabels,
} from 'messages/common';

import { comparatorOp, formatResources, formatScope, formatDeploymentExcludedScope } from './utils';

// JSON value name mapped to formatting for description page.
const fieldsMap = {
    id: {
        label: 'ID',
        formatValue: (value) => value,
    },
    name: {
        label: 'Name',
        formatValue: (value) => value,
    },
    lifecycleStages: {
        label: 'Lifecycle Stage',
        formatValue: (values) => values.map((v) => lifecycleStageLabels[v]).join(', '),
    },
    severity: {
        label: 'Severity',
        formatValue: (value) => {
            switch (value) {
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
        },
    },
    description: {
        label: 'Description',
        formatValue: (value) => value,
    },
    rationale: {
        label: 'Rationale',
        formatValue: (value) => value,
    },
    remediation: {
        label: 'Remediation',
        formatValue: (value) => value,
    },
    notifiers: {
        label: 'Notifications',
        formatValue: (values, props) =>
            props.notifiers
                .filter((notifier) => values.includes(notifier.id))
                .map((notifier) => notifier.name)
                .join(', '),
    },
    scope: {
        label: 'Restricted to Scopes',
        formatValue: (values, props) =>
            values && values.length ? values.map((scope) => formatScope(scope, props)) : null,
    },
    enforcementActions: {
        label: 'Enforcement Action',
        formatValue: (values) => values.map((v) => enforcementActionLabels[v]).join(', '),
    },
    disabled: {
        label: 'Enabled',
        formatValue: (value) => (!value ? 'Yes' : 'No'),
    },
    categories: {
        label: 'Categories',
        formatValue: (values) => values.join(', '),
    },
    exclusions: {
        label: 'Exclusions',
        formatValue: (values, props) => {
            const exclusionObj = {};
            const deploymentExcludedScopes = values
                .filter((obj) => obj.deployment && (obj.deployment.name || obj.deployment.scope))
                .map((obj) => obj.deployment);
            if (deploymentExcludedScopes.length > 0) {
                exclusionObj[
                    'Excluded Deployments'
                ] = deploymentExcludedScopes.map((deploymentExcludedScope) =>
                    formatDeploymentExcludedScope(deploymentExcludedScope, props)
                );
            }
            const images = values
                .filter((obj) => obj.image && obj.image.name !== '')
                .map((obj) => obj.image.name);
            if (images.length !== 0) {
                exclusionObj['Excluded Images'] = images;
            }
            return exclusionObj;
        },
    },
    imageName: {
        label: 'Image',
        formatValue: (value) => {
            const remote = value.remote ? `images named ${value.remote}` : 'any image';
            const tag = value.tag ? `tag ${value.tag}` : 'any tag';
            const registry = value.registry ? `registry ${value.registry}` : 'any registry';
            return `Alert on ${remote} using ${tag} from ${registry}`;
        },
    },
    imageAgeDays: {
        label: 'Days since image was created',
        formatValue: (value) => (value !== '0' ? `${Number(value)} Days ago` : ''),
    },
    noScanExists: {
        label: 'Image Scan Status',
        formatValue: () => 'Verify that the image is scanned',
    },
    scanAgeDays: {
        label: 'Days since image was last scanned',
        formatValue: (value) => (value !== '0' ? `${Number(value)} Days ago` : ''),
    },
    imageUser: {
        label: 'Image User',
        formatValue: (value) => value,
    },
    lineRule: {
        label: 'Dockerfile Line',
        formatValue: (value) => `${value.instruction} ${value.value}`,
    },
    cvss: {
        label: 'CVSS',
        formatValue: (value) => `${comparatorOp[value.op]} ${value.value}`,
    },
    cve: {
        label: 'CVE',
        formatValue: (value) => value,
    },
    fixedBy: {
        label: 'Fixed By',
        formatValue: (value) => value,
    },
    component: {
        label: 'Image Component',
        formatValue: (value) => {
            const name = value.name ? `${value.name}` : '';
            const version = value.version ? value.version : '';
            return `"${name}" with version "${version}"`;
        },
    },
    env: {
        label: 'Environment Variable',
        formatValue: (kvpolicy) => {
            const key = kvpolicy.key ? `${kvpolicy.key}` : '';
            const value = kvpolicy.value ? kvpolicy.value : '';
            const valueFrom = !kvpolicy.envVarSource
                ? ''
                : ` Value From: ${envVarSrcLabels[kvpolicy.envVarSource]}`;
            return `${key}=${value};${valueFrom}`;
        },
    },
    disallowedAnnotation: {
        label: 'Disallowed Annotation',
        formatValue: (kvpolicy) => {
            const key = kvpolicy.key ? `key=${kvpolicy.key}` : '';
            const value = kvpolicy.value ? `value=${kvpolicy.value}` : '';
            const comma = kvpolicy.key && kvpolicy.value ? ', ' : '';
            return `Alerts on deployments with the disallowed annotation ${key}${comma}${value}`;
        },
    },
    requiredLabel: {
        label: 'Required Label',
        formatValue: (kvpolicy) => {
            const key = kvpolicy.key ? `key=${kvpolicy.key}` : '';
            const value = kvpolicy.value ? `value=${kvpolicy.value}` : '';
            const comma = kvpolicy.key && kvpolicy.value ? ', ' : '';
            return `Alerts on deployments missing the required label ${key}${comma}${value}`;
        },
    },
    requiredAnnotation: {
        label: 'Required Annotation',
        formatValue: (kvpolicy) => {
            const key = kvpolicy.key ? `key=${kvpolicy.key}` : '';
            const value = kvpolicy.value ? `value=${kvpolicy.value}` : '';
            const comma = kvpolicy.key && kvpolicy.value ? ', ' : '';
            return `Alerts on deployments missing the required annotation ${key}${comma}${value}`;
        },
    },
    volumePolicy: {
        label: 'Volume Policy',
        formatValue: (value) => {
            const output = [];
            if (value.name) {
                output.push(`Name: ${value.name}`);
            }
            if (value.type) {
                output.push(`Type: ${value.type}`);
            }
            if (value.source) {
                output.push(`Source: ${value.source}`);
            }
            if (value.destination) {
                output.push(`Dest: ${value.destination}`);
            }
            output.push(value.readOnly ? 'Writable: No' : 'Writable: Yes');
            if (value.mountPropagation === 'HOST_TO_CONTAINER') {
                output.push(`Mount Propagation: Host to Container`);
            } else if (value.mountPropegation === 'BIDIRECTIONAL') {
                output.push(`Mount Propagation: Bidirectional`);
            } else {
                output.push(`Mount Propagation: None`);
            }
            return output.join(', ');
        },
    },
    nodePortPolicy: {
        label: 'Node Port',
        formatValue: (value) => {
            const protocol = value.protocol ? `${value.protocol} ` : '';
            const port = value.port ? value.port : '';
            return `${protocol}${port}`;
        },
    },
    portPolicy: {
        label: 'Port',
        formatValue: (value) => {
            const protocol = value.protocol ? `${value.protocol} ` : '';
            const port = value.port ? value.port : '';
            return `${protocol}${port}`;
        },
    },
    dropCapabilities: {
        label: 'Drop Capabilities',
        formatValue: (values) => values.join(', '),
    },
    addCapabilities: {
        label: 'Add Capabilities',
        formatValue: (values) => values.join(', '),
    },
    privileged: {
        label: 'Privileged',
        formatValue: (value) => (value ? 'Yes' : 'No'),
    },
    readOnlyRootFs: {
        label: 'Read Only Root Filesystem',
        formatValue: (value) => (value ? 'Yes' : 'Not Enabled'),
    },
    seccompProfileType: {
        label: 'Seccomp Profile Type',
        formatValue: (d) => {
            return `Seccomp profile type: ${
                seccompProfileTypeLabels[d.seccompProfileType] != null
                    ? seccompProfileTypeLabels[d.seccompProfileType]
                    : seccompProfileTypeLabels.RUNTIME_DEFAULT
            }`;
        },
    },
    HostPid: {
        label: 'Host PID',
        formatValue: (value) => (value ? 'Yes' : 'Not Enabled'),
    },
    containerResourcePolicy: {
        label: 'Container Resources',
        formatValue: formatResources,
    },
    processPolicy: {
        label: 'Process Execution',
        formatValue: (value) => {
            const name = value.name ? `Process matches name "${value.name}"` : 'Process';
            const args = value.args ? `and matches args "${value.args}"` : '';
            const ancestor = value.ancestor ? `and has ancestor matching "${value.ancestor}"` : '';
            const uid = value.uid ? `with uid ${value.uid}` : ``;
            return `${name} ${args} ${ancestor} ${uid}`;
        },
    },
    portExposurePolicy: {
        label: 'Port Exposure',
        formatValue: (value) => {
            const output = value.exposureLevels.map((element) => portExposureLabels[element]);
            return output.join(', ');
        },
    },
    hostMountPolicy: {
        label: 'Host Mount Policy',
        formatValue: (value) => (value.readOnly ? 'Not Enabled' : 'Writable: Yes'),
    },
    whitelistEnabled: {
        label: 'Excluded Scopes Enabled',
        formatValue: (value) => (value ? 'Yes' : 'No'),
    },
    permissionPolicy: {
        label: 'Minimum RBAC Permissions',
        formatValue: (value) => rbacPermissionLabels[value.permissionLevel],
    },
    requiredImageLabel: {
        label: 'Required Image Label',
        formatValue: (kvpolicy) => {
            const key = kvpolicy.key ? `key=${kvpolicy.key}` : '';
            const value = kvpolicy.value ? `value=${kvpolicy.value}` : '';
            const comma = kvpolicy.key && kvpolicy.value ? ', ' : '';
            return `Alerts on deployments with images missing the required label ${key}${comma}${value}`;
        },
    },
    disallowedImageLabel: {
        label: 'Disallowed Image Label',
        formatValue: (kvpolicy) => {
            const key = kvpolicy.key ? `key=${kvpolicy.key}` : '';
            const value = kvpolicy.value ? `value=${kvpolicy.value}` : '';
            const comma = kvpolicy.key && kvpolicy.value ? ', ' : '';
            return `Alerts on deployments with disallowed image label ${key}${comma}${value}`;
        },
    },
    HostIPC: {
        label: 'Host IPC',
        formatValue: (value) => (value ? 'Yes' : 'Not Enabled'),
    },
};

export default fieldsMap;
