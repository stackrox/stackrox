import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { ClipLoader as Loader } from 'react-spinners';

import TileContent from 'Components/TileContent';

type SummaryTileCountProps = {
    label: string;
    value?: number;
    loading?: boolean;
};

const SummaryTileCount = ({
    label,
    value = 0,
    loading = false,
}: SummaryTileCountProps): ReactElement => {
    return (
        <li
            key={label}
            className="flex flex-col px-3 lg:w-24 md:w-20 no-underline py-3 text-base-500 items-center justify-center font-condensed"
            data-testid="summary-tile-count"
        >
            {loading && !value ? (
                <Loader loading size={12} color="currentColor" />
            ) : (
                <TileContent
                    superText={value}
                    text={pluralize(label, value)}
                    textColorClass="text-base-500"
                />
            )}
        </li>
    );
};

export default SummaryTileCount;
