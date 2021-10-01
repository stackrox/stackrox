import React from 'react';

import RiskScore from './RiskScore';

export default {
    title: 'RiskScore',
    component: RiskScore,
};

export const basicRiskScore = () => {
    const score = 7;

    return <RiskScore score={score} />;
};
