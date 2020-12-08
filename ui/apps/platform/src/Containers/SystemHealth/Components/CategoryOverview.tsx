import React, { ReactElement } from 'react';

import { CategoryStyle, style0 } from '../utils/health';

type Props = {
    count: number;
    label: string;
    style: CategoryStyle;
};

const CategoryOverview = ({ count, label, style }: Props): ReactElement => {
    const { Icon } = style;
    const { fgColor } = count === 0 ? style0 : style;

    return (
        <div className={`flex justify-between leading-normal w-full ${fgColor}`}>
            <div className="flex">
                <Icon className="flex-shrink-0 h-4 w-4" />
                <span className="ml-2" data-testid="label">
                    {label}
                </span>
            </div>
            <span className="ml-2" data-testid="count">
                {count}
            </span>
        </div>
    );
};

export default CategoryOverview;
