import React from 'react';

import ClusterStatusButton from './ClusterStatusButton';

export default {
    title: 'ClusterStatusButton',
    component: ClusterStatusButton,
};

const context = 'border-base-400 border-r box-content h-16 text-base';

// TODO Why are the styles incorrect in tooltips?
const themeDark = 'theme-dark bg-base-100';
const themeLight = 'theme-light bg-base-200';

const style = { width: '49px' };

export const hasNeitherUnhealthyNorDegradedDarkMode = () => (
    <div className={`${context} ${themeDark}`} style={style}>
        <ClusterStatusButton degraded={0} unhealthy={0} />
    </div>
);

export const hasNeitherUnhealthyNorDegradedLightMode = () => (
    <div className={`${context} ${themeLight}`} style={style}>
        <ClusterStatusButton degraded={0} unhealthy={0} />
    </div>
);

export const hasDegradedButNotUnhealthyDarkMode = () => (
    <div className={`${context} ${themeDark}`} style={style}>
        <ClusterStatusButton degraded={1} unhealthy={0} />
    </div>
);

export const hasDegradedButNotUnhealthyLightMode = () => (
    <div className={`${context} ${themeLight}`} style={style}>
        <ClusterStatusButton degraded={1} unhealthy={0} />
    </div>
);

export const hasUnhealthyButNotDegradedDarkMode = () => (
    <div className={`${context} ${themeDark}`} style={style}>
        <ClusterStatusButton degraded={0} unhealthy={1} />
    </div>
);

export const hasUnhealthyButNotDegradedLightMode = () => (
    <div className={`${context} ${themeLight}`} style={style}>
        <ClusterStatusButton degraded={0} unhealthy={1} />
    </div>
);

export const hasBothUnhealthyAndDegradedAlignmentDarkMode = () => (
    <div className={`${context} ${themeDark}`} style={style}>
        <ClusterStatusButton degraded={4} unhealthy={13} />
    </div>
);

export const hasBothUnhealthyAndDegradedAlignmentLightMode = () => (
    <div className={`${context} ${themeLight}`} style={style}>
        <ClusterStatusButton degraded={4} unhealthy={13} />
    </div>
);
