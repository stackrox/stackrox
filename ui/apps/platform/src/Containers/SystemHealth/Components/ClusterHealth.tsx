import React, { ReactElement } from 'react';

import CategoryOverview from './CategoryOverview';

import {
    CountableText,
    CountMap,
    LabelMap,
    StyleMap,
    getCountableText,
    getProblemStyle,
    style0,
} from '../utils/health';

type Props = {
    countMap: CountMap;
    healthyKey: string;
    healthySubtext: string;
    healthyText: CountableText;
    labelMap: LabelMap;
    problemText: CountableText;
    styleMap: StyleMap;
};

const ClusterHealth = ({
    countMap,
    healthyKey,
    healthySubtext,
    healthyText,
    labelMap,
    problemText,
    styleMap,
}: Props): ReactElement => {
    // Assume countMap does not have status which are neither healthy nor problem.
    const healthyCount = countMap[healthyKey] ?? 0;
    const problemCount =
        Object.values(countMap).reduce((total, value) => total + value, 0) - healthyCount;

    if (problemCount !== 0) {
        const { bgColor, fgColor } = getProblemStyle(countMap, healthyKey, styleMap);

        return (
            <div className={`flex flex-col justify-between w-full ${bgColor}`}>
                <div className={fgColor}>
                    <div className="flex justify-center mb-2 mt-4">
                        <span className="leading-none text-4xl" data-testid="problem-count">
                            {problemCount}
                        </span>
                    </div>
                    <div className="leading-normal px-2 text-center" data-testid="problem-text">
                        {getCountableText(problemText, problemCount)}
                    </div>
                </div>
                <ul className="p-1 w-full">
                    {Object.keys(countMap)
                        .filter((key) => key !== healthyKey && countMap[key] !== 0)
                        .map((key) => (
                            <li className="p-1" key={key} data-testid={key}>
                                <CategoryOverview
                                    count={countMap[key]}
                                    label={labelMap[key]}
                                    style={styleMap[key]}
                                />
                            </li>
                        ))}
                </ul>
            </div>
        );
    }

    const { Icon, fgColor } = healthyCount === 0 ? style0 : styleMap[healthyKey];
    const text = `${healthyCount} ${getCountableText(healthyText, healthyCount)}`;

    return (
        <div className="flex flex-col justify-between w-full">
            <div className={fgColor}>
                <div className="flex justify-center mb-2 mt-4">
                    <Icon className="h-6 w-6" />
                </div>
                <div className="leading-normal px-2 text-center" data-testid="healthy-text">
                    {text}
                </div>
            </div>
            {healthyCount !== 0 && (
                <div className="leading-normal p-2">
                    <span className="italic" data-testid="healthy-subtext">
                        {healthySubtext}
                    </span>
                </div>
            )}
        </div>
    );
};

export default ClusterHealth;
