import React from 'react';

import Row from './Row';

const Rows = ({ data, translateX, translateY }) => {
    return (
        <g transform={`translate(${translateX}, ${translateY})`}>
            {data.map((datum, index) => {
                const { id, name, events } = datum;
                const isOddRow = index % 2 !== 0;
                const rowHeight = 48;
                return (
                    <Row
                        key={id}
                        name={name}
                        events={events}
                        isOdd={isOddRow}
                        height={rowHeight}
                        width="100%"
                        translateX={0}
                        translateY={index * rowHeight}
                    />
                );
            })}
        </g>
    );
};

export default Rows;
